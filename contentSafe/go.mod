module github.com/23233/ggg/contentSafe

go 1.24.0

replace (
	github.com/23233/ggg/logger => ../logger
	github.com/23233/ggg/sv => ../sv
	github.com/23233/ggg/ut => ../ut
)

require (
	github.com/23233/ggg/logger v0.0.0-20250521015733-ba3370e5e495
	github.com/23233/ggg/sv v0.0.0-20250521015733-ba3370e5e495
	github.com/23233/ggg/ut v0.0.0-20250521015733-ba3370e5e495
	github.com/23233/lad v0.1.3
	github.com/bluele/gcache v0.0.2
	github.com/go-creed/sat v1.0.3
	github.com/imroc/req/v3 v3.52.1
	github.com/kataras/iris/v12 v12.2.11
	github.com/pkg/errors v0.9.1
	github.com/schollz/progressbar/v3 v3.18.0
)

require (
	github.com/BurntSushi/toml v1.5.0 // indirect
	github.com/CloudyKit/fastprinter v0.0.0-20200109182630-33d98a066a53 // indirect
	github.com/CloudyKit/jet/v6 v6.3.1 // indirect
	github.com/Joker/jade v1.1.3 // indirect
	github.com/OneOfOne/xxhash v1.2.8 // indirect
	github.com/Shopify/goreferrer v0.0.0-20250513162709-b78e2829e40b // indirect
	github.com/andybalholm/brotli v1.1.1 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/cloudflare/circl v1.6.1 // indirect
	github.com/colduction/randomizer v1.4.0 // indirect
	github.com/fatih/structs v1.1.0 // indirect
	github.com/flosch/pongo2/v4 v4.0.2 // indirect
	github.com/gabriel-vasile/mimetype v1.4.9 // indirect
	github.com/gaukas/godicttls v0.0.3 // indirect
	github.com/getsentry/sentry-go v0.33.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.26.0 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/gomarkdown/markdown v0.0.0-20250311123330-531bef5e742b // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20250501235452-c0086092b71a // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/css v1.0.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/iancoleman/strcase v0.3.0 // indirect
	github.com/icholy/digest v1.1.0 // indirect
	github.com/iris-contrib/schema v0.0.6 // indirect
	github.com/itchyny/base58-go v0.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/jxskiss/base62 v1.1.0 // indirect
	github.com/kataras/blocks v0.0.11 // indirect
	github.com/kataras/golog v0.1.13 // indirect
	github.com/kataras/pio v0.0.14 // indirect
	github.com/kataras/sitemap v0.0.6 // indirect
	github.com/kataras/tunnel v0.0.4 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible // indirect
	github.com/lestrrat-go/strftime v1.1.0 // indirect
	github.com/mailgun/raymond/v2 v2.0.48 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/microcosm-cc/bluemonday v1.0.27 // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/onsi/ginkgo/v2 v2.23.4 // indirect
	github.com/quic-go/qpack v0.5.1 // indirect
	github.com/quic-go/qtls-go1-19 v0.3.2 // indirect
	github.com/quic-go/qtls-go1-20 v0.2.2 // indirect
	github.com/quic-go/quic-go v0.52.0 // indirect
	github.com/redis/rueidis v1.0.60 // indirect
	github.com/redis/rueidis/rueidiscompat v1.0.60 // indirect
	github.com/refraction-networking/utls v1.7.3 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/schollz/closestmatch v2.1.0+incompatible // indirect
	github.com/shockerli/cvt v0.2.8 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/tdewolff/minify/v2 v2.23.6 // indirect
	github.com/tdewolff/parse/v2 v2.8.1 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/yosssi/ace v0.0.5 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	go.mongodb.org/mongo-driver v1.17.3 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/automaxprocs v1.6.0 // indirect
	go.uber.org/mock v0.5.2 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/crypto v0.38.0 // indirect
	golang.org/x/exp v0.0.0-20250506013437-ce4c2cf36ca6 // indirect
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sync v0.14.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/term v0.32.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	golang.org/x/time v0.11.0 // indirect
	golang.org/x/tools v0.33.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gorm.io/gorm v1.26.1 // indirect
)
