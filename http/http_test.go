package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/Genesic/mixednuts/http/middleware"
	"github.com/gorilla/mux"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMuxServer_Serve(t *testing.T) {
	ctx := context.Background()
	app := newApp(1234)

	go func(ctx context.Context, server *MuxServer) {
		err := server.Serve(ctx)
		So(err, ShouldBeNil)
	}(ctx, app)

	client := NewClient("localhost:1234").
		WithHeaders("tracker", "yes").
		WithTimeout(time.Second)
	Convey("test http app and client", t, func() {
		Convey("test app ping pong", func() {
			headers := map[string]string{
				"validator": "yes",
			}
			resp := new(http.Header)
			code, err := client.CommonDoWithJSON(http.MethodGet, "/header", headers, nil, resp)
			So(err, ShouldBeNil)
			So(code, ShouldEqual, http.StatusOK)
			So(resp.Get("tracker"), ShouldResemble, "yes")
			So(resp.Get("validator"), ShouldResemble, "yes")
			So(resp.Get("Content-Type"), ShouldResemble, "application/json")
		})

		Convey("test client timeout", func() {
			code, err := client.CommonDoWithJSON(http.MethodGet, "/timeout", nil, nil, nil)
			So(err.Error(), ShouldEndWith, "(Client.Timeout exceeded while awaiting headers)")
			So(code, ShouldEqual, 0)
		})

		Convey("test form", func() {
			form := url.Values{}
			form.Add("trackers", "first")
			form.Add("trackers", "second")
			code, result, err := client.CommonDoWithForm(http.MethodPost, "/form", nil, form)
			fmt.Println(string(result))
			resp := url.Values{}
			_ = json.Unmarshal(result, &resp)
			So(err, ShouldBeNil)
			So(code, ShouldEqual, http.StatusOK)
			So(resp, ShouldResemble, form)

		})
	})
}

func pingHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	})
}

func newApp(port int) *MuxServer {

	controllers := []Controller{
		&mockController{wait: 2 * time.Second},
	}

	app := NewMuxServer(port, Config{}).
		WithMiddlewares(
			middleware.RequestIDMiddleware,
			middleware.ResponseMiddleware,
			middleware.LogMiddleware(),
			middleware.RequestDurationMiddleware,
		).
		WithControllers(controllers...).
		WithAdditionalHandlers(
			"/ping", pingHandler(),
		)

	return app
}

type mockController struct {
	headerKey string
	wait      time.Duration
}

func (m *mockController) RegisterHandlers(router *mux.Router) {
	router.
		Methods(http.MethodGet).
		Path("/header").
		Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(r.Header)
		}))

	router.
		Methods(http.MethodGet).
		Path("/timeout").
		Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(m.wait)
			w.WriteHeader(http.StatusOK)
		}))

	router.
		Methods(http.MethodPost).
		Path("/form").
		Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := r.ParseForm(); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(r.Form)
		}))
}
