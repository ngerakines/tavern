module github.com/ngerakines/tavern

go 1.13

require (
	github.com/gin-contrib/gzip v0.0.1
	github.com/gin-contrib/zap v0.0.0-20190528085758-3cc18cd8fce3
	github.com/gin-gonic/gin v1.4.0
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/jinzhu/gorm v1.9.10
	github.com/kr/pretty v0.1.0 // indirect
	github.com/lib/pq v1.2.0
	github.com/oklog/run v1.0.0
	github.com/piprate/json-gold v0.2.0
	github.com/pkg/errors v0.8.1
	github.com/urfave/cli v1.21.0
	go.uber.org/atomic v1.4.0 // indirect
	go.uber.org/zap v1.10.0
)

replace github.com/ugorji/go v1.1.4 => github.com/ugorji/go/codec v0.0.0-20190204201341-e444a5086c43
