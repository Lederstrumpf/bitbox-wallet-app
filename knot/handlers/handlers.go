package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/shiftdevices/godbb/dbbdevice"
	deviceHandlers "github.com/shiftdevices/godbb/dbbdevice/handlers"
	"github.com/shiftdevices/godbb/deterministicwallet"
	walletHandlers "github.com/shiftdevices/godbb/deterministicwallet/handlers"
	"github.com/shiftdevices/godbb/knot"
	"github.com/shiftdevices/godbb/knot/binweb"
	"github.com/shiftdevices/godbb/util/jsonp"
	qrcode "github.com/skip2/go-qrcode"
)

type KnotInterface interface {
	XPub() (string, error)
	OnWalletInit(f func(deterministicwallet.Interface))
	OnWalletUninit(f func())
	OnDeviceInit(f func(dbbdevice.Interface))
	OnDeviceUninit(f func())
	Start() <-chan knot.Event
}

type Handlers struct {
	Router *mux.Router
	knot   KnotInterface
	// apiPort is the port on which this API will run. It is fed into the static javascript app
	// that is served, so the client knows where to connect to.
	apiPort           int
	knotEvents        <-chan knot.Event
	websocketUpgrader websocket.Upgrader
}

func NewHandlers(
	knot KnotInterface,
	apiPort int,
) *Handlers {
	router := mux.NewRouter()
	handlers := &Handlers{
		Router:  router,
		knot:    knot,
		apiPort: apiPort,
		websocketUpgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		knotEvents: knot.Start(),
	}

	getApiRouter := func(subrouter *mux.Router) func(string, func(*http.Request) (interface{}, error)) *mux.Route {
		return func(path string, f func(*http.Request) (interface{}, error)) *mux.Route {
			return subrouter.HandleFunc(path, apiMiddleware(f))
		}

	}

	apiRouter := router.PathPrefix("/api").Subrouter()
	apiHandleFunc := getApiRouter(apiRouter)
	apiRouter.HandleFunc("/qr", handlers.getQRCode).Methods("GET")
	apiHandleFunc("/xpub", handlers.getXPubHandler).Methods("GET")

	walletHandlers_ := walletHandlers.NewHandlers(
		getApiRouter(apiRouter.PathPrefix("/wallet/btc").Subrouter()),
	)
	knot.OnWalletInit(func(wallet deterministicwallet.Interface) {
		walletHandlers_.Init(wallet)
	})
	knot.OnWalletUninit(func() {
		walletHandlers_.Uninit()
	})

	deviceHandlers_ := deviceHandlers.NewHandlers(
		getApiRouter(apiRouter.PathPrefix("/device").Subrouter()),
	)
	knot.OnDeviceInit(func(device dbbdevice.Interface) {
		deviceHandlers_.Init(device)
	})
	knot.OnDeviceUninit(func() {
		deviceHandlers_.Uninit()
	})

	apiRouter.HandleFunc("/events", handlers.eventsHandler)

	// Serve static files for the UI.
	router.Handle("/{rest:.*}",
		http.FileServer(&assetfs.AssetFS{
			Asset: func(name string) ([]byte, error) {
				body, err := binweb.Asset(name)
				if err != nil {
					return nil, err
				}
				if regexp.MustCompile(`^bundle\..+\.js$`).MatchString(name) {
					// TODO: move function elsewhere, use the template package.
					body = bytes.Replace(body, []byte("{{ API_PORT }}"), []byte(fmt.Sprintf("%d", handlers.apiPort)), -1)
				}
				return body, nil
			},
			AssetDir:  binweb.AssetDir,
			AssetInfo: binweb.AssetInfo,
			Prefix:    "",
		}))

	return handlers
}

func writeJSON(w http.ResponseWriter, value interface{}) {
	if err := json.NewEncoder(w).Encode(value); err != nil {
		panic(err)
	}
}

func (handlers *Handlers) getQRCode(w http.ResponseWriter, r *http.Request) {
	data := r.URL.Query().Get("data")
	qr, err := qrcode.New(data, qrcode.Medium)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	qr.Write(256, w)
}

func (handlers *Handlers) getXPubHandler(r *http.Request) (interface{}, error) {
	return handlers.knot.XPub()
}

func (handlers *Handlers) eventsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := handlers.websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		panic(err)
	}

	sendChan, quitChan := runWebsocket(conn)
	go func() {
		for {
			select {
			case <-quitChan:
				return
			default:
				select {
				case <-quitChan:
					return
				case event := <-handlers.knotEvents:
					sendChan <- jsonp.MustMarshal(map[string]string{
						"type": event.Type,
						"data": event.Data,
					})
				}
			}
		}
	}()
}

func apiMiddleware(h func(*http.Request) (interface{}, error)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/json")
		// This enables us to run a server on a different port serving just the UI, while still
		// allowing it to access the API.
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8080")
		value, err := h(r)
		if err != nil {
			writeJSON(w, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, value)
	}
}
