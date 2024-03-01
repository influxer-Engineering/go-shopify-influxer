go mod tidy

git commit -am "go-shopify-influxer v1.0.1"
git push origin v1.0.1

GOPROXY=proxy.golang.org go list -m github.com/influxer-Engineering.com/go-shopify-influxer@v1.0.1
