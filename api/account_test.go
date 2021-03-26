package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	mockdb "github.com/xiusl/bank/db/mock"
	db "github.com/xiusl/bank/db/sqlc"
	"github.com/xiusl/bank/util"
)

func TestGetAccountAPI(t *testing.T) {
	account := randomAccount()
	testCases := []struct {
		name         string
		accountID    int64
		buildStuds   func(store *mockdb.MockStore)
		exceptStatus int
	}{
		{
			name:      "OK",
			accountID: account.ID,
			buildStuds: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					Return(account, nil).
					Times(1)
			},
			exceptStatus: http.StatusOK,
		},
		{
			name:      "Not Found",
			accountID: account.ID,
			buildStuds: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					Return(db.Account{}, sql.ErrNoRows).
					Times(1)
			},
			exceptStatus: http.StatusNotFound,
		},
		{
			name:      "InternalError",
			accountID: account.ID,
			buildStuds: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					Return(db.Account{}, sql.ErrConnDone).
					Times(1)
			},
			exceptStatus: http.StatusInternalServerError,
		},
		{
			name:      "InvalidID",
			accountID: 0,
			buildStuds: func(store *mockdb.MockStore) {
				const id int64 = 0
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Any()).
					Times(0)
			},
			exceptStatus: http.StatusBadRequest,
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStuds(store)

			server := NewServer(store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/accounts/%d", tc.accountID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)
			require.Equal(t, tc.exceptStatus, recorder.Code)
		})
	}
}

func randomAccount() db.Account {
	return db.Account{
		ID:        util.RandomInt(1, 1000),
		Owner:     util.RandomOwner(),
		Balance:   util.RandomMoney(),
		Currency:  util.RandomCurrency(),
		CreatedAt: time.Now(),
	}
}
