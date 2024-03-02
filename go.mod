module github.com/influxer-Engineering/go-shopify-influxer

go 1.21

retract (
	v1.0.2 // testing publish issue
	v1.0.1 // premature
	v1.0.0 // outdated
)

require (
	github.com/google/go-querystring v1.0.0
	github.com/jarcoal/httpmock v1.3.0
	github.com/shopspring/decimal v0.0.0-20200105231215-408a2507e114
)
