package file

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
type GenerateURLRequest struct {
	FileName string `json:"fileName"`
}

// GenerateURLResponse 是获取上传链接API成功时的响应体结构
type GenerateURLResponse struct {
	UploadURL string `json:"uploadUrl"`
	ObjectKey string `json:"objectKey"`
	AccessURL string `json:"accessUrl"`
}

// New 创建一个新的图床客户端实例
// host: 你的 Workers 服务地址, 例如 "https://your-worker.workers.dev"
// authKey: 你的认证密钥 (X-Auth-Key)
func New(host, authKey string) *Client {
	return &Client{
		host:    host,
		authKey: authKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second, // 设置一个合理的超时时间
		},
	}
}

// GetUploadURL 从 Workers 服务获取一个预签名的上传URL和最终的访问URL
// ctx: 上下文，用于控制请求的取消或超时
// fileName: 你希望在服务器上保存的文件名
func (c *Client) GetUploadURL(ctx context.Context, fileName string) (*GenerateURLResponse, error) {
	// 1. 准备请求体
	reqBody := GenerateURLRequest{FileName: fileName}
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

// UploadFile 是一个高级方法，封装了获取URL并上传文件的整个过程
// ctx: 上下文，用于控制请求的取消或超时
// fileName: 你希望在服务器上保存的文件名
// fileContent: 文件的内容，需要是一个 io.Reader
func (c *Client) UploadFile(ctx context.Context, fileName string, fileContent io.Reader) (string, error) {
	// 1. 获取预签名上传链接
	uploadInfo, err := c.GetUploadURL(ctx, fileName)
	if err != nil {
		return "", fmt.Errorf("获取上传链接失败: %w", err)
	}

	// 2. 创建用于上传文件的 HTTP PUT 请求
	// 注意：这里使用的是获取到的 uploadInfo.UploadURL
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadInfo.UploadURL, fileContent)
	if err != nil {
		return "", fmt.Errorf("创建文件上传请求失败: %w", err)
	}
	// 注意：上传到R2的预签名URL时，通常不需要设置额外的认证头，
	// 因为所有认证信息都在URL的查询参数里了。
	// 如果你的文件需要指定Content-Type，可以在这里设置。
	// req.Header.Set("Content-Type", "image/png") // 例如

	// 3. 执行上传
	uploadResp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("上传文件到R2失败: %w", err)
	}
	defer uploadResp.Body.Close()

	// 4. 检查上传是否成功
	if uploadResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(uploadResp.Body)
		return "", fmt.Errorf("上传文件到R2返回错误, 状态码: %d, 响应: %s", uploadResp.StatusCode, string(bodyBytes))
	}

	// 5. 返回公开访问的URL
	return uploadInfo.AccessURL, nil
}
