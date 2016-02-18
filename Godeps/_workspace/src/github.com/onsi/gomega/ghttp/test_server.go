/*
Package ghttp supports testing HTTP clients by providing a test server (simply a thin wrapper around httptest's server) that supports
registering multiple handlers.  Incoming requests are not routed between the different handlers
- rather it is merely the order of the handlers that matters.  The first request is handled by the first
registered handler, the second request by the second handler, etc.

The intent here is to have each handler *verify* that the incoming request is valid.  To accomplish, ghttp
also provides a collection of bite-size handlers that each perform one aspect of request verification.  These can
be composed together and registered with a ghttp server.  The result is an expressive language for describing
the requests generated by the client under test.

Here's a simple example, note that the server handler is only defined in one BeforeEach and then modified, as required, by the nested BeforeEaches.
A more comprehensive example is available at https://onsi.github.io/gomega/#_testing_http_clients

	var _ = Describe("A Sprockets Client", func() {
		var server *ghttp.Server
		var client *SprocketClient
		BeforeEach(func() {
			server = ghttp.NewServer()
			client = NewSprocketClient(server.URL(), "skywalker", "tk427")
		})

		AfterEach(func() {
			server.Close()
		})

		Describe("fetching sprockets", func() {
			var statusCode int
			var sprockets []Sprocket
			BeforeEach(func() {
				statusCode = http.StatusOK
				sprockets = []Sprocket{}
				server.AppendHandlers(ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/sprockets"),
					ghttp.VerifyBasicAuth("skywalker", "tk427"),
					ghttp.RespondWithJSONEncodedPtr(&statusCode, &sprockets),
				))
			})

			Context("when requesting all sprockets", func() {
				Context("when the response is succesful", func() {
					BeforeEach(func() {
						sprockets = []Sprocket{
							NewSprocket("Alfalfa"),
							NewSprocket("Banana"),
						}
					})

					It("should return the returned sprockets", func() {
						Ω(client.Sprockets()).Should(Equal(sprockets))
					})
				})

				Context("when the response is missing", func() {
					BeforeEach(func() {
						statusCode = http.StatusNotFound
					})

					It("should return an empty list of sprockets", func() {
						Ω(client.Sprockets()).Should(BeEmpty())
					})
				})

				Context("when the response fails to authenticate", func() {
					BeforeEach(func() {
						statusCode = http.StatusUnauthorized
					})

					It("should return an AuthenticationError error", func() {
						sprockets, err := client.Sprockets()
						Ω(sprockets).Should(BeEmpty())
						Ω(err).Should(MatchError(AuthenticationError))
					})
				})

				Context("when the response is a server failure", func() {
					BeforeEach(func() {
						statusCode = http.StatusInternalServerError
					})

					It("should return an InternalError error", func() {
						sprockets, err := client.Sprockets()
						Ω(sprockets).Should(BeEmpty())
						Ω(err).Should(MatchError(InternalError))
					})
				})
			})

			Context("when requesting some sprockets", func() {
				BeforeEach(func() {
					sprockets = []Sprocket{
						NewSprocket("Alfalfa"),
						NewSprocket("Banana"),
					}

					server.WrapHandler(0, ghttp.VerifyRequest("GET", "/sprockets", "filter=FOOD"))
				})

				It("should make the request with a filter", func() {
					Ω(client.Sprockets("food")).Should(Equal(sprockets))
				})
			})
		})
	})
*/
package ghttp

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"sync"

	. "github.com/cloudfoundry/mkman/Godeps/_workspace/src/github.com/onsi/gomega"
)

func new() *Server {
	return &Server{
		AllowUnhandledRequests:     false,
		UnhandledRequestStatusCode: http.StatusInternalServerError,
		writeLock:                  &sync.Mutex{},
	}
}

type routedHandler struct {
	method     string
	pathRegexp *regexp.Regexp
	path       string
	handler    http.HandlerFunc
}

// NewServer returns a new `*ghttp.Server` that wraps an `httptest` server.  The server is started automatically.
func NewServer() *Server {
	s := new()
	s.HTTPTestServer = httptest.NewServer(s)
	return s
}

// NewUnstartedServer return a new, unstarted, `*ghttp.Server`.  Useful for specifying a custom listener on `server.HTTPTestServer`.
func NewUnstartedServer() *Server {
	s := new()
	s.HTTPTestServer = httptest.NewUnstartedServer(s)
	return s
}

// NewTLSServer returns a new `*ghttp.Server` that wraps an `httptest` TLS server.  The server is started automatically.
func NewTLSServer() *Server {
	s := new()
	s.HTTPTestServer = httptest.NewTLSServer(s)
	return s
}

type Server struct {
	//The underlying httptest server
	HTTPTestServer *httptest.Server

	//Defaults to false.  If set to true, the Server will allow more requests than there are registered handlers.
	AllowUnhandledRequests bool

	//The status code returned when receiving an unhandled request.
	//Defaults to http.StatusInternalServerError.
	//Only applies if AllowUnhandledRequests is true
	UnhandledRequestStatusCode int

	receivedRequests []*http.Request
	requestHandlers  []http.HandlerFunc
	routedHandlers   []routedHandler

	writeLock *sync.Mutex
	calls     int
}

//Start() starts an unstarted ghttp server.  It is a catastrophic error to call Start more than once (thanks, httptest).
func (s *Server) Start() {
	s.HTTPTestServer.Start()
}

//URL() returns a url that will hit the server
func (s *Server) URL() string {
	return s.HTTPTestServer.URL
}

//Close() should be called at the end of each test.  It spins down and cleans up the test server.
func (s *Server) Close() {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()

	server := s.HTTPTestServer
	s.HTTPTestServer = nil
	server.Close()
}

//ServeHTTP() makes Server an http.Handler
//When the server receives a request it handles the request in the following order:
//
//1. If the request matches a handler registered with RouteToHandler, that handler is called.
//2. Otherwise, if there are handlers registered via AppendHandlers, those handlers are called in order.
//3. If all registered handlers have been called then:
//   a) If AllowUnhandledRequests is true, the request will be handled with response code of UnhandledRequestStatusCode
//   b) If AllowUnhandledRequests is false, the request will not be handled and the current test will be marked as failed.
func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.writeLock.Lock()
	defer func() {
		recover()
	}()

	s.receivedRequests = append(s.receivedRequests, req)
	if routedHandler, ok := s.handlerForRoute(req.Method, req.URL.Path); ok {
		s.writeLock.Unlock()
		routedHandler(w, req)
	} else if s.calls < len(s.requestHandlers) {
		h := s.requestHandlers[s.calls]
		s.calls++
		s.writeLock.Unlock()
		h(w, req)
	} else {
		s.writeLock.Unlock()
		if s.AllowUnhandledRequests {
			ioutil.ReadAll(req.Body)
			req.Body.Close()
			w.WriteHeader(s.UnhandledRequestStatusCode)
		} else {
			Ω(req).Should(BeNil(), "Received Unhandled Request")
		}
	}
}

//ReceivedRequests is an array containing all requests received by the server (both handled and unhandled requests)
func (s *Server) ReceivedRequests() []*http.Request {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()

	return s.receivedRequests
}

//RouteToHandler can be used to register handlers that will always handle requests that match
//the passed in method and path.
//
//The path may be either a string object or a *regexp.Regexp.
func (s *Server) RouteToHandler(method string, path interface{}, handler http.HandlerFunc) {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()

	rh := routedHandler{
		method:  method,
		handler: handler,
	}

	switch p := path.(type) {
	case *regexp.Regexp:
		rh.pathRegexp = p
	case string:
		rh.path = p
	default:
		panic("path must be a string or a regular expression")
	}

	for i, existingRH := range s.routedHandlers {
		if existingRH.method == method &&
			reflect.DeepEqual(existingRH.pathRegexp, rh.pathRegexp) &&
			existingRH.path == rh.path {
			s.routedHandlers[i] = rh
			return
		}
	}
	s.routedHandlers = append(s.routedHandlers, rh)
}

func (s *Server) handlerForRoute(method string, path string) (http.HandlerFunc, bool) {
	for _, rh := range s.routedHandlers {
		if rh.method == method {
			if rh.pathRegexp != nil {
				if rh.pathRegexp.Match([]byte(path)) {
					return rh.handler, true
				}
			} else if rh.path == path {
				return rh.handler, true
			}
		}
	}

	return nil, false
}

//AppendHandlers will appends http.HandlerFuncs to the server's list of registered handlers.  The first incoming request is handled by the first handler, the second by the second, etc...
func (s *Server) AppendHandlers(handlers ...http.HandlerFunc) {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()

	s.requestHandlers = append(s.requestHandlers, handlers...)
}

//SetHandler overrides the registered handler at the passed in index with the passed in handler
//This is useful, for example, when a server has been set up in a shared context, but must be tweaked
//for a particular test.
func (s *Server) SetHandler(index int, handler http.HandlerFunc) {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()

	s.requestHandlers[index] = handler
}

//GetHandler returns the handler registered at the passed in index.
func (s *Server) GetHandler(index int) http.HandlerFunc {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()

	return s.requestHandlers[index]
}

//WrapHandler combines the passed in handler with the handler registered at the passed in index.
//This is useful, for example, when a server has been set up in a shared context but must be tweaked
//for a particular test.
//
//If the currently registered handler is A, and the new passed in handler is B then
//WrapHandler will generate a new handler that first calls A, then calls B, and assign it to index
func (s *Server) WrapHandler(index int, handler http.HandlerFunc) {
	existingHandler := s.GetHandler(index)
	s.SetHandler(index, CombineHandlers(existingHandler, handler))
}

func (s *Server) CloseClientConnections() {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()

	s.HTTPTestServer.CloseClientConnections()
}
