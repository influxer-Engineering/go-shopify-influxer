go mod tidy

git commit -am "go-shopify-influxer v1.0.3"
git tag v1.0.3
git push origin v1.0.3

GOPROXY=proxy.golang.org go list -m github.com/Influxer-Engineering/go-shopify-influxer@v1.0.3
