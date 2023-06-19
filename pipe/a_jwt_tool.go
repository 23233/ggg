package pipe

import (
	"context"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	irisJwt "github.com/iris-contrib/middleware/jwt"
	"github.com/kataras/iris/v12"
	"github.com/redis/rueidis"
	"strings"
	"time"
)

var (
	jwtSecret = []byte("HefNcCJPz2eT7rq2eW7L9WaFLYO4zZO4446gr")
)

var jwtConfig = irisJwt.Config{
	//Extractor : jwtToken.FromParameter("token")
	//Extractor : jwtToken.FromAuthHeader // default
	ErrorHandler: func(ctx iris.Context, err error) {
		if err == nil {
			return
		}
		ctx.StopExecution()
		ctx.StatusCode(iris.StatusUnauthorized)
		_ = ctx.JSON(iris.Map{
			"detail": err.Error(),
		})
	},
	ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	},
	Expiration:          false,
	CredentialsOptional: false,
	SigningMethod:       jwt.SigningMethodHS256,
}

var jwtParser = new(jwt.Parser)

var ctJwt = irisJwt.New(jwtConfig)

const (
	JwtPrefix      = "Bearer "
	JwtShortPrefix = "Short "
	JwtShortLen    = 12
)

type JwtHelper struct {
	rdb rueidis.Client
}

// TokenExtract jwt验证
func (c *JwtHelper) TokenExtract(token string, m *irisJwt.Middleware) (map[string]any, error) {
	var tk = token
	if strings.HasPrefix(token, JwtPrefix) {
		tk = strings.TrimPrefix(token, JwtPrefix)
	}
	parsedToken, err := jwtParser.Parse(tk, m.Config.ValidationKeyGetter)
	if err != nil {
		return nil, err
	}

	if m.Config.SigningMethod != nil && m.Config.SigningMethod.Alg() != parsedToken.Header["alg"] {
		err := fmt.Errorf("expected %s signing method but token specified %s",
			m.Config.SigningMethod.Alg(),
			parsedToken.Header["alg"])
		return nil, err
	}

	// Check if the parsed token is valid...
	if !parsedToken.Valid {
		return nil, irisJwt.ErrTokenInvalid
	}

	if m.Config.Expiration {
		if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok {
			if expired := claims.VerifyExpiresAt(time.Now().Unix(), true); !expired {
				return nil, irisJwt.ErrTokenExpired
			}
		}
	}
	if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok {
		return claims, nil
	}

	return nil, irisJwt.ErrTokenInvalid
}

func (c *JwtHelper) GenJwtToken(userId, env string) string {
	n := time.Now()
	mcp := jwt.MapClaims{
		"userId":    userId,
		"env":       env,
		"loginTime": n.Format(time.RFC3339),
	}
	token := irisJwt.NewTokenWithClaims(jwt.SigningMethodHS256, mcp)
	tokenString, _ := token.SignedString(jwtSecret)
	return tokenString
}

func (c *JwtHelper) JwtShortRedisGenKey(shortToken string) string {
	return "short:" + shortToken
}
func (c *JwtHelper) JwtRedisGenKey(userId, env string) string {
	return "jwt:" + userId + ":" + env
}

func (c *JwtHelper) JwtRedisGetKey(ctx context.Context, key string) rueidis.RedisResult {
	return c.rdb.Do(ctx, c.rdb.B().Get().Key(key).Build())
}

func (c *JwtHelper) JwtSaveToken(ctx context.Context, key string, token string, expired time.Duration) error {
	insetResp := c.rdb.Do(ctx, c.rdb.B().Set().Key(key).Value(token).ExSeconds(int64(expired.Seconds())).Build())
	return insetResp.Error()
}

func NewJwtHelper(rdb rueidis.Client) *JwtHelper {
	return &JwtHelper{
		rdb: rdb,
	}
}
