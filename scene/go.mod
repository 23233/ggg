module github.com/23233/ggg/scene

go 1.21

replace (
	github.com/23233/ggg/logger => ../logger
	github.com/23233/ggg/pipe => ../pipe
	github.com/23233/ggg/sv => ../sv
	github.com/23233/ggg/ut => ../ut
)

require (
	github.com/23233/ggg/pipe v0.0.0-20231126023434-6c801150988b
	github.com/23233/ggg/ut v0.0.0-20231125085718-c7312d0c4e06
	github.com/23233/jsonschema v0.11.2
	github.com/kataras/iris/v12 v12.2.10
	github.com/kataras/realip v0.0.2
	github.com/pkg/errors v0.9.1
	github.com/qiniu/qmgo v1.1.7
	github.com/redis/rueidis v1.0.25
	github.com/stretchr/testify v1.8.4
)

require (
	github.com/23233/ggg/contentSafe v0.0.0-20230628084524-a5a98249dc81 // indirect
	github.com/23233/ggg/logger v0.0.0-20240126064458-4ef52b984a5e // indirect
	github.com/23233/ggg/sv v0.0.0-20221002105326-9e8ed7cbb14f // indirect
	github.com/23233/lad v0.1.3 // indirect
	github.com/23233/user_agent v0.1.1 // indirect
	github.com/BurntSushi/toml v1.3.2 // indirect
	github.com/CloudyKit/fastprinter v0.0.0-20200109182630-33d98a066a53 // indirect
	github.com/CloudyKit/jet/v6 v6.2.0 // indirect
	github.com/Joker/jade v1.1.3 // indirect
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible // indirect
	github.com/OneOfOne/xxhash v1.2.8 // indirect
	github.com/Shopify/goreferrer v0.0.0-20220729165902-8cddb4f5de06 // indirect
	github.com/andybalholm/brotli v1.1.0 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/bluele/gcache v0.0.2 // indirect
	github.com/casbin/casbin/v2 v2.71.1 // indirect
	github.com/casbin/redis-adapter/v2 v2.4.0 // indirect
	github.com/colduction/randomizer v1.0.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fatih/structs v1.1.0 // indirect
	github.com/flosch/pongo2/v4 v4.0.2 // indirect
	github.com/gabriel-vasile/mimetype v1.4.2 // indirect
	github.com/gaukas/godicttls v0.0.3 // indirect
	github.com/getsentry/sentry-go v0.13.0 // indirect
	github.com/go-creed/sat v1.0.3 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.14.1 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.0 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/gomarkdown/markdown v0.0.0-20231222211730-1d6d20845b47 // indirect
	github.com/gomodule/redigo v1.8.9 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/pprof v0.0.0-20230602150820-91b7bce49751 // indirect
	github.com/google/uuid v1.5.0 // indirect
	github.com/gookit/filter v1.1.4 // indirect
	github.com/gookit/goutil v0.5.15 // indirect
	github.com/gookit/validate v1.4.6 // indirect
	github.com/gorilla/css v1.0.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/iancoleman/orderedmap v0.2.0 // indirect
	github.com/iancoleman/strcase v0.2.0 // indirect
	github.com/imroc/req/v3 v3.37.2 // indirect
	github.com/iris-contrib/go.uuid v2.0.0+incompatible // indirect
	github.com/iris-contrib/middleware/jwt v0.0.0-20230531125531-980d3a09a458 // indirect
	github.com/iris-contrib/schema v0.0.6 // indirect
	github.com/itchyny/base58-go v0.2.1 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/jxskiss/base62 v1.1.0 // indirect
	github.com/kataras/blocks v0.0.8 // indirect
	github.com/kataras/golog v0.1.11 // indirect
	github.com/kataras/pio v0.0.13 // indirect
	github.com/kataras/sitemap v0.0.6 // indirect
	github.com/kataras/tunnel v0.0.4 // indirect
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible // indirect
	github.com/lestrrat-go/strftime v1.0.5 // indirect
	github.com/mailgun/raymond/v2 v2.0.48 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/microcosm-cc/bluemonday v1.0.26 // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/onsi/ginkgo/v2 v2.12.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/quic-go/qpack v0.4.0 // indirect
	github.com/quic-go/qtls-go1-19 v0.3.2 // indirect
	github.com/quic-go/qtls-go1-20 v0.2.2 // indirect
	github.com/quic-go/quic-go v0.35.1 // indirect
	github.com/refraction-networking/utls v1.3.2 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.0 // indirect
	github.com/schollz/closestmatch v2.1.0+incompatible // indirect
	github.com/schollz/progressbar/v3 v3.13.1 // indirect
	github.com/shockerli/cvt v0.2.7 // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	github.com/sony/sonyflake v1.1.0 // indirect
	github.com/tdewolff/minify/v2 v2.20.14 // indirect
	github.com/tdewolff/parse/v2 v2.7.8 // indirect
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common v1.0.689 // indirect
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms v1.0.689 // indirect
	github.com/ulule/limiter/v3 v3.11.2 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/yosssi/ace v0.0.5 // indirect
	github.com/youmark/pkcs8 v0.0.0-20201027041543-1326539a0a0a // indirect
	go.mongodb.org/mongo-driver v1.11.7 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.21.0 // indirect
	golang.org/x/crypto v0.18.0 // indirect
	golang.org/x/exp v0.0.0-20240112132812-db7319d0e0e3 // indirect
	golang.org/x/mod v0.14.0 // indirect
	golang.org/x/net v0.20.0 // indirect
	golang.org/x/sync v0.6.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
	golang.org/x/term v0.16.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/tools v0.17.0 // indirect
	google.golang.org/protobuf v1.32.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
