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
)

func TestPaymentsTransactionsList(t *testing.T) {
	setup()
	defer teardown()

	httpmock.RegisterResponder("GET", fmt.Sprintf("https://fooshop.myshopify.com/%s/shopify_payments/balance/transactions.json", client.pathPrefix),
		httpmock.NewBytesResponder(200, loadFixture("payments_transactions.json")))
	date1 := OnlyDate{time.Date(2013, 11, 0o1, 0, 0, 0, 0, time.UTC)}
	paymentsTransactions, err := client.PaymentsTransactions.List(context.Background(), PaymentsTransactionsListOptions{PayoutId: 623721858})
	if err != nil {
		t.Errorf("PaymentsTransactions.List returned error: %v", err)
	}

	expected := []PaymentsTransactions{
		{
			Id:                       699519475,
			Type:                     PaymentsTransactionsDebit,
			Test:                     false,
			PayoutId:                 623721858,
			PayoutStatus:             PayoutStatusPaid,
			Currency:                 "USD",
			Amount:                   "-50.00",
			Fee:                      "0.00",
			Net:                      "-50.00",
			SourceId:                 460709370,
			SourceType:               "adjustment",
			SourceOrderId:            0,
			SourceOrderTransactionId: 0,
			ProcessedAt:              date1,
		},
		{
			Id:                       77412310,
			Type:                     PaymentsTransactionsCredit,
			Test:                     false,
			PayoutId:                 623721858,
			PayoutStatus:             PayoutStatusPaid,
			Currency:                 "USD",
			Amount:                   "50.00",
			Fee:                      "0.00",
			Net:                      "50.00",
			SourceId:                 374511569,
			SourceType:               "Payments::Balance::AdjustmentReversal",
			SourceOrderId:            0,
			SourceOrderTransactionId: 0,
			ProcessedAt:              date1,
		},
		{
			Id:                       1006917261,
			Type:                     PaymentsTransactionsRefund,
			Test:                     false,
			PayoutId:                 623721858,
			PayoutStatus:             PayoutStatusPaid,
			Currency:                 "USD",
			Amount:                   "-3.45",
			Fee:                      "0.00",
			Net:                      "-3.45",
			SourceId:                 1006917261,
			SourceType:               "Payments::Refund",
			SourceOrderId:            217130470,
			SourceOrderTransactionId: 1006917261,
			ProcessedAt:              date1,
		},
	}
	if !reflect.DeepEqual(paymentsTransactions, expected) {
		t.Errorf("PaymentsTransactions.List returned %+v, expected %+v", paymentsTransactions, expected)
	}
}

func TestPaymentsTransactionsListIncorrectDate(t *testing.T) {
	setup()
	defer teardown()

	httpmock.RegisterResponder("GET", fmt.Sprintf("https://fooshop.myshopify.com/%s/shopify_payments/balance/transactions.json", client.pathPrefix),
		httpmock.NewStringResponder(200, `{"transactions": [{"id":1, "processed_at":"20-02-2"}]}`))

	date1 := OnlyDate{time.Date(2022, 0o2, 0o3, 0, 0, 0, 0, time.Local)}
	_, err := client.PaymentsTransactions.List(context.Background(), PaymentsTransactionsListOptions{ProcessedAt: &date1})
	if err == nil {
		t.Errorf("PaymentsTransactions.List returned success, expected error: %v", err)
	}
}

func TestPaymentsTransactionsListError(t *testing.T) {
	setup()
	defer teardown()

	httpmock.RegisterResponder("GET", fmt.Sprintf("https://fooshop.myshopify.com/%s/shopify_payments/balance/transactions.json", client.pathPrefix),
		httpmock.NewStringResponder(500, ""))

	expectedErrMessage := "Unknown Error"

	paymentsTransactions, err := client.PaymentsTransactions.List(context.Background(), nil)
	if paymentsTransactions != nil {
		t.Errorf("PaymentsTransactions.List returned transactions, expected nil: %v", err)
	}

	if err == nil || err.Error() != expectedErrMessage {
		t.Errorf("PaymentsTransactions.List err returned %+v, expected %+v", err, expectedErrMessage)
	}
}

func TestPaymentsTransactionsListAll(t *testing.T) {
	setup()
	defer teardown()

	listURL := fmt.Sprintf("https://fooshop.myshopify.com/%s/shopify_payments/balance/transactions.json", client.pathPrefix)

	cases := []struct {
		name                          string
		expectedPaymentsTransactionss []PaymentsTransactions
		expectedRequestURLs           []string
		expectedLinkHeaders           []string
		expectedBodies                []string
		expectedErr                   error
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
				`{"transactions": [{"id":1},{"id":2}]}`,
				`{"transactions": [{"id":3},{"id":4}]}`,
			},
			expectedPaymentsTransactionss: []PaymentsTransactions{{Id: 1}, {Id: 2}, {Id: 3}, {Id: 4}},
			expectedErr:                   nil,
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
				`{"transactions": [{"id":1}]}`,
			},
			expectedPaymentsTransactionss: []PaymentsTransactions{{Id: 1}},
			expectedErr:                   nil,
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
				`{"transactions": []}`,
			},
			expectedPaymentsTransactionss: []PaymentsTransactions{},
			expectedErr:                   errors.New("page_info is missing"),
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

			transactions, err := client.PaymentsTransactions.ListAll(context.Background(), nil)
			if !reflect.DeepEqual(transactions, c.expectedPaymentsTransactionss) {
				t.Errorf("test %d PaymentsTransactions.ListAll orders returned %+v, expected %+v", i, transactions, c.expectedPaymentsTransactionss)
			}

			if (c.expectedErr != nil || err != nil) && err.Error() != c.expectedErr.Error() {
				t.Errorf(
					"test %d PaymentsTransactions.ListAll err returned %+v, expected %+v",
					i,
					err,
					c.expectedErr,
				)
			}
		})
	}
}

func TestPaymentsTransactionsListWithPagination(t *testing.T) {
	setup()
	defer teardown()

	listURL := fmt.Sprintf("https://fooshop.myshopify.com/%s/shopify_payments/balance/transactions.json", client.pathPrefix)
	date1 := OnlyDate{time.Date(2013, 11, 0o1, 0, 0, 0, 0, time.UTC)}

	cases := []struct {
		body                         string
		linkHeader                   string
		expectedPaymentsTransactions []PaymentsTransactions
		expectedPagination           *Pagination
		expectedErr                  error
	}{
		// Expect empty pagination when there is no link header
		{
			string(loadFixture("payments_transactions.json")),
			"",
			[]PaymentsTransactions{
				{
					Id:                       699519475,
					Type:                     PaymentsTransactionsDebit,
					Test:                     false,
					PayoutId:                 623721858,
					PayoutStatus:             PayoutStatusPaid,
					Currency:                 "USD",
					Amount:                   "-50.00",
					Fee:                      "0.00",
					Net:                      "-50.00",
					SourceId:                 460709370,
					SourceType:               "adjustment",
					SourceOrderId:            0,
					SourceOrderTransactionId: 0,
					ProcessedAt:              date1,
				},
				{
					Id:                       77412310,
					Type:                     PaymentsTransactionsCredit,
					Test:                     false,
					PayoutId:                 623721858,
					PayoutStatus:             PayoutStatusPaid,
					Currency:                 "USD",
					Amount:                   "50.00",
					Fee:                      "0.00",
					Net:                      "50.00",
					SourceId:                 374511569,
					SourceType:               "Payments::Balance::AdjustmentReversal",
					SourceOrderId:            0,
					SourceOrderTransactionId: 0,
					ProcessedAt:              date1,
				},
				{
					Id:                       1006917261,
					Type:                     PaymentsTransactionsRefund,
					Test:                     false,
					PayoutId:                 623721858,
					PayoutStatus:             PayoutStatusPaid,
					Currency:                 "USD",
					Amount:                   "-3.45",
					Fee:                      "0.00",
					Net:                      "-3.45",
					SourceId:                 1006917261,
					SourceType:               "Payments::Refund",
					SourceOrderId:            217130470,
					SourceOrderTransactionId: 1006917261,
					ProcessedAt:              date1,
				},
			},
			new(Pagination),
			nil,
		},
		// Invalid link header responses
		{
			"{}",
			"invalid link",
			[]PaymentsTransactions(nil),
			nil,
			ResponseDecodingError{Message: "could not extract pagination link header"},
		},
		{
			"{}",
			`<:invalid.url>; rel="next"`,
			[]PaymentsTransactions(nil),
			nil,
			ResponseDecodingError{Message: "pagination does not contain a valid URL"},
		},
		{
			"{}",
			`<http://valid.url?%invalid_query>; rel="next"`,
			[]PaymentsTransactions(nil),
			nil,
			errors.New(`invalid URL escape "%in"`),
		},
		{
			"{}",
			`<http://valid.url>; rel="next"`,
			[]PaymentsTransactions(nil),
			nil,
			ResponseDecodingError{Message: "page_info is missing"},
		},
		{
			"{}",
			`<http://valid.url?page_info=foo&limit=invalid>; rel="next"`,
			[]PaymentsTransactions(nil),
			nil,
			errors.New(`strconv.Atoi: parsing "invalid": invalid syntax`),
		},
		// Valid link header responses
		{
			`{"transactions": [{"id":1}]}`,
			`<http://valid.url?page_info=foo&limit=2>; rel="next"`,
			[]PaymentsTransactions{{Id: 1}},
			&Pagination{
				NextPageOptions: &ListOptions{PageInfo: "foo", Limit: 2},
			},
			nil,
		},
		{
			`{"transactions": [{"id":2}]}`,
			`<http://valid.url?page_info=foo>; rel="next", <http://valid.url?page_info=bar>; rel="previous"`,
			[]PaymentsTransactions{{Id: 2}},
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

		paymentsTransactions, pagination, err := client.PaymentsTransactions.ListWithPagination(context.Background(), nil)
		if !reflect.DeepEqual(paymentsTransactions, c.expectedPaymentsTransactions) {
			t.Errorf("test %d PaymentsTransactions.ListWithPagination transactions returned %+v, expected %+v", i, paymentsTransactions, c.expectedPaymentsTransactions)
		}

		if !reflect.DeepEqual(pagination, c.expectedPagination) {
			t.Errorf(
				"test %d PaymentsTransactions.ListWithPagination pagination returned %+v, expected %+v",
				i,
				pagination,
				c.expectedPagination,
			)
		}

		if (c.expectedErr != nil || err != nil) && err.Error() != c.expectedErr.Error() {
			t.Errorf(
				"test %d PaymentsTransactions.ListWithPagination err returned %+v, expected %+v",
				i,
				err,
				c.expectedErr,
			)
		}
	}
}

func TestPaymentsTransactionsGet(t *testing.T) {
	setup()
	defer teardown()

	httpmock.RegisterResponder("GET", fmt.Sprintf("https://fooshop.myshopify.com/%s/shopify_payments/balance/transactions/623721858.json", client.pathPrefix),
		httpmock.NewBytesResponder(200, loadFixture("payments_transaction.json")))

	paymentsTransactions, err := client.PaymentsTransactions.Get(context.Background(), 623721858, nil)
	if err != nil {
		t.Errorf("PaymentsTransactions.Get returned error: %v", err)
	}
	date1 := OnlyDate{time.Date(2013, 11, 0o1, 0, 0, 0, 0, time.UTC)}

	expected := &PaymentsTransactions{
		Id:                       699519475,
		Type:                     PaymentsTransactionsDebit,
		Test:                     false,
		PayoutId:                 623721858,
		PayoutStatus:             PayoutStatusPaid,
		Currency:                 "USD",
		Amount:                   "-50.00",
		Fee:                      "0.00",
		Net:                      "-50.00",
		SourceId:                 460709370,
		SourceType:               "adjustment",
		SourceOrderId:            0,
		SourceOrderTransactionId: 0,
		ProcessedAt:              date1,
	}
	if !reflect.DeepEqual(paymentsTransactions, expected) {
		t.Errorf("PaymentsTransactions.Get returned %+v, expected %+v", paymentsTransactions, expected)
	}
}
