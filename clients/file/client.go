package file

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// Client 是与图床 Workers 服务交互的客户端
type Client struct {
	host       string
	authKey    string
	httpClient *http.Client
}

// GenerateURLRequest 是获取上传链接API的请求体结构
// 我们添加了可选的过期时间和最大文件大小字段
type GenerateURLRequest struct {
	FileName         string `json:"fileName"`
	ExpiresInSeconds int    `json:"expiresInSeconds,omitempty"` // omitempty 使得当值为0时，该字段不会被序列化
	MaxSizeMb        int    `json:"maxSizeMb,omitempty"`        // 同上
}

// GenerateURLResponse 是获取上传链接API成功时的响应体结构
// 响应结构已完全更新以适配 Presigned POST
type GenerateURLResponse struct {
	PostURL   string            `json:"postUrl"`   // 上传的目标 URL
	FormData  map[string]string `json:"formData"`  // 上传时需要携带的表单字段
	ObjectKey string            `json:"objectKey"` // 文件在 R2 中的 Key
	AccessURL string            `json:"accessUrl"` // 文件的公开访问 URL
}

// UploadOptions 封装了上传时的所有可配置项
type UploadOptions struct {
	FileName         string    // 必需：文件名
	FileContent      io.Reader // 必需：文件内容
	MaxSizeMb        int       // 可选：限制本次上传的最大文件大小 (MB)，0 表示使用服务器默认值
	ExpiresInSeconds int       // 可选：链接的有效时间 (秒)，0 表示使用服务器默认值
}

// New 创建一个新的图床客户端实例
// host: 你的 Workers 服务地址, 例如 "https://your-worker.workers.dev"
// authKey: 你的认证密钥 (X-Auth-Key)
func New(host, authKey string) *Client {
	return &Client{
		host:    host,
		authKey: authKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // 增加超时以应对更大的文件上传
		},
	}
}

// GetUploadURL 从 Workers 服务获取一个预签名的上传表单数据
// ctx: 上下文，用于控制请求的取消或超时
// fileName: 你希望在服务器上保存的文件名
// maxSizeMb: 本次上传允许的最大文件大小(MB)，传 0 表示使用服务器默认值
// expiresInSeconds: 签名有效时长(秒)，传 0 表示使用服务器默认值
func (c *Client) GetUploadURL(ctx context.Context, fileName string, maxSizeMb, expiresInSeconds int) (*GenerateURLResponse, error) {
	// 1. 准备请求体
	reqBody := GenerateURLRequest{
		FileName:         fileName,
		MaxSizeMb:        maxSizeMb,
		ExpiresInSeconds: expiresInSeconds,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

	// 2. 创建 HTTP POST 请求
	endpoint := c.host + "/generate-upload-url"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 3. 设置必要的请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Key", c.authKey)

	// 4. 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 5. 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API返回错误, 状态码: %d, 响应: %s", resp.StatusCode, string(bodyBytes))
	}

	// 6. 解析成功的JSON响应
	var apiResponse GenerateURLResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("解析API响应失败: %w", err)
	}

	return &apiResponse, nil
}

// UploadFile 是一个简化的方法，使用服务器的默认设置上传文件
// ctx: 上下文，用于控制请求的取消或超时
// fileName: 你希望在服务器上保存的文件名
// fileContent: 文件的内容，需要是一个 io.Reader
func (c *Client) UploadFile(ctx context.Context, fileName string, fileContent io.Reader) (string, error) {
	options := UploadOptions{
		FileName:    fileName,
		FileContent: fileContent,
		// MaxSizeMb 和 ExpiresInSeconds 均为 0, 将使用服务器的默认值
	}
	return c.UploadFileWithOptions(ctx, options)
}

// UploadFileWithOptions 是一个功能更全的方法，封装了获取URL并上传文件的整个过程
// ctx: 上下文，用于控制请求的取消或超时
// options: 包含文件名、文件内容和可选参数的结构体
func (c *Client) UploadFileWithOptions(ctx context.Context, options UploadOptions) (string, error) {
	// 1. 获取预签名上传链接和表单数据
	uploadInfo, err := c.GetUploadURL(ctx, options.FileName, options.MaxSizeMb, options.ExpiresInSeconds)
	if err != nil {
		return "", fmt.Errorf("获取上传链接失败: %w", err)
	}

	// 2. 创建 multipart/form-data 请求体
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// 3. 将从 API 获取的 formData 字段写入表单
	for key, value := range uploadInfo.FormData {
		if err := writer.WriteField(key, value); err != nil {
			return "", fmt.Errorf("写入表单字段 %s 失败: %w", key, err)
		}
	}

	// 4. 创建文件字段并写入文件内容
	// R2/S3 Presigned Post 要求 'file' 字段是最后一个字段
	part, err := writer.CreateFormFile("file", options.FileName)
	if err != nil {
		return "", fmt.Errorf("创建文件表单部分失败: %w", err)
	}
	if _, err := io.Copy(part, options.FileContent); err != nil {
		return "", fmt.Errorf("向表单写入文件内容失败: %w", err)
	}

	// 5. 关闭 multipart writer，这会写入结尾的 boundary
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("关闭 multipart writer 失败: %w", err)
	}

	// 6. 创建用于上传文件的 HTTP POST 请求
	// 注意：URL 是 uploadInfo.PostURL
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadInfo.PostURL, &requestBody)
	if err != nil {
		return "", fmt.Errorf("创建文件上传请求失败: %w", err)
	}

	// 7. 设置正确的 Content-Type，它包含了 multipart 的 boundary
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// 8. 执行上传
	uploadResp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("上传文件到R2失败: %w", err)
	}
	defer uploadResp.Body.Close()

	// 9. 检查上传是否成功 (R2/S3 成功时通常返回 204 No Content)
	if uploadResp.StatusCode < 200 || uploadResp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(uploadResp.Body)
		return "", fmt.Errorf("上传文件到R2返回错误, 状态码: %d, 响应: %s", uploadResp.StatusCode, string(bodyBytes))
	}

	// 10. 返回公开访问的URL
	return uploadInfo.AccessURL, nil
}
