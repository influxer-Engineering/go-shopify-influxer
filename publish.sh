go mod tidy

git commit -am "go-shopify-influxer v1.0.2"
git tag v1.0.2
git push origin v1.0.2

GOPROXY=proxy.golang.org go list -m github.com/influxer-Engineering.com/go-shopify-influxer@v1.0.2
