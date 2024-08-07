package smab

import (
	"github.com/iris-contrib/middleware/jwt"
	"github.com/kataras/iris/v12"
	"time"
)

var MySecret = []byte("8657684ae02840ead423e0d781a7a885")

// CustomJwt 自定义JWT
// 使用办法 中间层 handler.CustomJwt.Serve, handler.TokenToUserUidMiddleware,user handler
var CustomJwt = jwt.New(jwt.Config{
	ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
		return MySecret, nil
	},
	Expiration:    true,
	SigningMethod: jwt.SigningMethodHS256,
})

// TokenToUserUidMiddleware 登录token存储信息 记录到上下文中
func TokenToUserUidMiddleware(ctx iris.Context) {
	user := ctx.Values().Get(CustomJwt.Config.ContextKey).(*jwt.Token)
	jwtData := user.Claims.(jwt.MapClaims)
	userUid, err := jwtData["userUid"].(string)
	if err != true {
		ctx.StatusCode(iris.StatusUnauthorized)
		_ = ctx.JSON(iris.Map{
			"detail": "jwt token if fail",
		})
		return
	}
	userName, err := jwtData["userName"].(string)
	if err != true {
		ctx.StatusCode(iris.StatusUnauthorized)
		_ = ctx.JSON(iris.Map{
			"detail": "jwt token if fail",
		})
		return
	}
	// 这里可以遍历所有的token信息
	//for key, value := range jwtData {
	//	_, _ = ctx.Writef("%s = %s", key, value)
	//}

	ctx.Values().Set("uid", userUid)
	ctx.Values().Set("un", userName)
	ctx.Next() // execute the next handler, in this case the main one.
}

// GenJwtToken 生成token
func GenJwtToken(userUid, userName string) string {
	token := jwt.NewTokenWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userUid":  userUid,
		"userName": userName,
		"exp":      time.Now().Add(time.Hour * 120).Unix(), //过期时间 120小时
	})
	tokenString, _ := token.SignedString(MySecret)
	return tokenString
}
