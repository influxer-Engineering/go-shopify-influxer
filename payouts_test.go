package goshopify

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/shopspring/decimal"
)

func TestPayoutsList(t *testing.T) {
	setup()
	defer teardown()

	httpmock.RegisterResponder("GET", fmt.Sprintf("https://fooshop.myshopify.com/%s/shopify_payments/payouts.json", client.pathPrefix),
		httpmock.NewBytesResponder(200, loadFixture("payouts_filtered.json")))

	date1 := OnlyDate{time.Date(2013, 11, 0o1, 0, 0, 0, 0, time.UTC)}
	payouts, err := client.Payouts.List(context.Background(), PayoutsListOptions{Date: &date1})
	if err != nil {
		t.Errorf("Payouts.List returned error: %v", err)
	}

	expected := []Payout{{Id: 854088011, Date: date1, Currency: "USD", Amount: decimal.NewFromFloat(43.12), Status: PayoutStatusScheduled}}
	if !reflect.DeepEqual(payouts, expected) {
		t.Errorf("Payouts.List returned %+v, expected %+v", payouts, expected)
	}
}

func TestPayoutsListIncorrectDate(t *testing.T) {
	setup()
	defer teardown()

	httpmock.RegisterResponder("GET", fmt.Sprintf("https://fooshop.myshopify.com/%s/shopify_payments/payouts.json", client.pathPrefix),
		httpmock.NewStringResponder(200, `{"payouts": [{"id":1, "date":"20-02-2"}]}`))

	date1 := OnlyDate{time.Date(2022, 0o2, 0o3, 0, 0, 0, 0, time.Local)}
	_, err := client.Payouts.List(context.Background(), PayoutsListOptions{Date: &date1})
	if err == nil {
		t.Errorf("Payouts.List returned success, expected error: %v", err)
	}
}

func TestPayoutsListError(t *testing.T) {
	setup()
	defer teardown()

	httpmock.RegisterResponder("GET", fmt.Sprintf("https://fooshop.myshopify.com/%s/shopify_payments/payouts.json", client.pathPrefix),
		httpmock.NewStringResponder(500, ""))

	expectedErrMessage := "Unknown Error"

	payouts, err := client.Payouts.List(context.Background(), nil)
	if payouts != nil {
		t.Errorf("Payouts.List returned payouts, expected nil: %v", err)
	}

	if err == nil || err.Error() != expectedErrMessage {
		t.Errorf("Payouts.List err returned %+v, expected %+v", err, expectedErrMessage)
	}
}

func TestPayoutListAll(t *testing.T) {
	setup()
	defer teardown()

	listURL := fmt.Sprintf("https://fooshop.myshopify.com/%s/shopify_payments/payouts.json", client.pathPrefix)

	cases := []struct {
		name                string
		expectedPayouts     []Payout
		expectedRequestURLs []string
		expectedLinkHeaders []string
		expectedBodies      []string
		expectedErr         error
	}{
		{
			name: "Pulls the next page",
			expectedRequestURLs: []string{
				listURL,
				fmt.Sprintf("%s?page_info=pg2", listURL),
			},
			expectedLinkHeaders: []string{
				`<http://valid.url?page_info=pg2>; rel="next"`,
				`<http://valid.url?page_info=pg1>; rel="previous"`,
			},
			expectedBodies: []string{
				`{"payouts": [{"id":1},{"id":2}]}`,
				`{"payouts": [{"id":3},{"id":4}]}`,
			},
			expectedPayouts: []Payout{{Id: 1}, {Id: 2}, {Id: 3}, {Id: 4}},
			expectedErr:     nil,
		},
		{
			name: "Stops when there is not a next page",
			expectedRequestURLs: []string{
				listURL,
			},
			expectedLinkHeaders: []string{
				`<http://valid.url?page_info=pg2>; rel="previous"`,
			},
			expectedBodies: []string{
				`{"payouts": [{"id":1}]}`,
			},
			expectedPayouts: []Payout{{Id: 1}},
			expectedErr:     nil,
		},
		{
			name: "Returns errors when required",
			expectedRequestURLs: []string{
				listURL,
			},
			expectedLinkHeaders: []string{
				`<http://valid.url?paage_info=pg2>; rel="previous"`,
			},
			expectedBodies: []string{
				`{"payouts": []}`,
			},
			expectedPayouts: []Payout{},
			expectedErr:     errors.New("page_info is missing"),
		},
	}

	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if len(c.expectedRequestURLs) != len(c.expectedLinkHeaders) {
				t.Errorf(
					"test case must have the same number of expected request urls (%d) as expected link headers (%d)",
					len(c.expectedRequestURLs),
					len(c.expectedLinkHeaders),
				)

				return
			}

			if len(c.expectedRequestURLs) != len(c.expectedBodies) {
				t.Errorf(
					"test case must have the same number of expected request urls (%d) as expected bodies (%d)",
					len(c.expectedRequestURLs),
					len(c.expectedBodies),
				)

				return
			}

			for i := range c.expectedRequestURLs {
				response := &http.Response{
					StatusCode: 200,
					Body:       httpmock.NewRespBodyFromString(c.expectedBodies[i]),
					Header: http.Header{
						"Link": {c.expectedLinkHeaders[i]},
					},
				}

				httpmock.RegisterResponder("GET", c.expectedRequestURLs[i], httpmock.ResponderFromResponse(response))
			}

			payouts, err := client.Payouts.ListAll(context.Background(), nil)
			if !reflect.DeepEqual(payouts, c.expectedPayouts) {
				t.Errorf("test %d Payout.ListAll orders returned %+v, expected %+v", i, payouts, c.expectedPayouts)
			}

			if (c.expectedErr != nil || err != nil) && err.Error() != c.expectedErr.Error() {
				t.Errorf(
					"test %d Payout.ListAll err returned %+v, expected %+v",
					i,
					err,
					c.expectedErr,
				)
			}
		})
	}
}

func TestPayoutsListWithPagination(t *testing.T) {
	setup()
	defer teardown()

	listURL := fmt.Sprintf("https://fooshop.myshopify.com/%s/shopify_payments/payouts.json", client.pathPrefix)

	cases := []struct {
		body               string
		linkHeader         string
		expectedPayouts    []Payout
		expectedPagination *Pagination
		expectedErr        error
	}{
		// Expect empty pagination when there is no link header
		{
			string(loadFixture("payouts.json")),
			"",
			[]Payout{
				{Id: 854088011, Date: OnlyDate{time.Date(2013, 11, 1, 0, 0, 0, 0, time.UTC)}, Currency: "USD", Amount: decimal.NewFromFloat(43.12), Status: PayoutStatusScheduled},
				{Id: 512467833, Date: OnlyDate{time.Date(2013, 11, 1, 0, 0, 0, 0, time.UTC)}, Currency: "USD", Amount: decimal.NewFromFloat(43.12), Status: PayoutStatusFailed},
			},
			new(Pagination),
			nil,
		},
		// Invalid link header responses
		{
			"{}",
			"invalid link",
			[]Payout(nil),
			nil,
			ResponseDecodingError{Message: "could not extract pagination link header"},
		},
		{
			"{}",
			`<:invalid.url>; rel="next"`,
			[]Payout(nil),
			nil,
			ResponseDecodingError{Message: "pagination does not contain a valid URL"},
		},
		{
			"{}",
			`<http://valid.url?%invalid_query>; rel="next"`,
			[]Payout(nil),
			nil,
			errors.New(`invalid URL escape "%in"`),
		},
		{
			"{}",
			`<http://valid.url>; rel="next"`,
			[]Payout(nil),
			nil,
			ResponseDecodingError{Message: "page_info is missing"},
		},
		{
			"{}",
			`<http://valid.url?page_info=foo&limit=invalid>; rel="next"`,
			[]Payout(nil),
			nil,
			errors.New(`strconv.Atoi: parsing "invalid": invalid syntax`),
		},
		// Valid link header responses
		{
			`{"payouts": [{"id":1}]}`,
			`<http://valid.url?page_info=foo&limit=2>; rel="next"`,
			[]Payout{{Id: 1}},
			&Pagination{
				NextPageOptions: &ListOptions{PageInfo: "foo", Limit: 2},
			},
			nil,
		},
		{
			`{"payouts": [{"id":2}]}`,
			`<http://valid.url?page_info=foo>; rel="next", <http://valid.url?page_info=bar>; rel="previous"`,
			[]Payout{{Id: 2}},
			&Pagination{
				NextPageOptions:     &ListOptions{PageInfo: "foo"},
				PreviousPageOptions: &ListOptions{PageInfo: "bar"},
			},
			nil,
		},
	}
	for i, c := range cases {
		response := &http.Response{
			StatusCode: 200,
			Body:       httpmock.NewRespBodyFromString(c.body),
			Header: http.Header{
				"Link": {c.linkHeader},
			},
		}

		httpmock.RegisterResponder("GET", listURL, httpmock.ResponderFromResponse(response))

		payouts, pagination, err := client.Payouts.ListWithPagination(context.Background(), nil)
		if !reflect.DeepEqual(payouts, c.expectedPayouts) {
			t.Errorf("test %d Payouts.ListWithPagination payouts returned %+v, expected %+v", i, payouts, c.expectedPayouts)
		}

		if !reflect.DeepEqual(pagination, c.expectedPagination) {
			t.Errorf(
				"test %d Payouts.ListWithPagination pagination returned %+v, expected %+v",
				i,
				pagination,
				c.expectedPagination,
			)
		}

		if (c.expectedErr != nil || err != nil) && err.Error() != c.expectedErr.Error() {
			t.Errorf(
				"test %d Payouts.ListWithPagination err returned %+v, expected %+v",
				i,
				err,
				c.expectedErr,
			)
		}
	}
}

func TestPayoutsGet(t *testing.T) {
	setup()
	defer teardown()

	httpmock.RegisterResponder("GET", fmt.Sprintf("https://fooshop.myshopify.com/%s/shopify_payments/payouts/623721858.json", client.pathPrefix),
		httpmock.NewBytesResponder(200, loadFixture("payout.json")))

	payout, err := client.Payouts.Get(context.Background(), 623721858, nil)
	if err != nil {
		t.Errorf("Payouts.Get returned error: %v", err)
	}

	expected := &Payout{
		Id:       623721858,
		Date:     OnlyDate{time.Date(2012, 11, 12, 0, 0, 0, 0, time.UTC)},
		Status:   PayoutStatusPaid,
		Currency: "USD",
		Amount:   decimal.NewFromFloat(41.9),
	}
	if !reflect.DeepEqual(payout, expected) {
		t.Errorf("Payouts.Get returned %+v, expected %+v", payout, expected)
	}
}
