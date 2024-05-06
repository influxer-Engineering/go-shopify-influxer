go mod tidy

git commit -am "go-shopify-influxer v1.0.5"
git tag v1.0.5
git push origin v1.0.5

GOPROXY=proxy.golang.org go list -m github.com/influxer-Engineering/go-shopify-influxer@v1.0.5
