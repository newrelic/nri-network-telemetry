package httpserver

import (
	"fmt"
	"net/http"
	"os"

	newrelic "github.com/newrelic/go-agent"
	log "github.com/sirupsen/logrus"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type ControlMessage int

const (
	ControlMessageQuit  ControlMessage = iota
	ControlMessageStart ControlMessage = iota
	ControlMessageReady ControlMessage = iota
	ControlMessageDone  ControlMessage = iota
)

type Server struct {
	version string
	address string
	port    int
	handler http.Handler
	nr      newrelic.Application
}

/******************************************************************************
 *
 * Create a new HTTPserver instance
 *
 ******************************************************************************/
func New(version string, address string, port int, nr newrelic.Application) *Server {
	return (&Server{
		version: version,
		address: address,
		port:    port,
		nr:      nr,
	}).initializeHandler()
}

/******************************************************************************
 *
 * Initialize HTTP handlers
 *
 ******************************************************************************/
func (s *Server) initializeHandler() *Server {
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc(newrelic.WrapHandleFunc(s.nr, "/", s.rootHandler)).Methods("GET")
	router.HandleFunc(newrelic.WrapHandleFunc(s.nr, "/version", s.versionHandler)).Methods("GET")
	router.HandleFunc(newrelic.WrapHandleFunc(s.nr, "/status/check", s.statusCheckHandler)).Methods("GET")

	// Wrap all requests with the logging handler (apache-like logs)
	s.handler = handlers.LoggingHandler(os.Stdout, router)

	return s
}

/******************************************************************************
 *
 * HTTP handler functions
 *
 ******************************************************************************/
func (s *Server) rootHandler(rw http.ResponseWriter, r *http.Request) {
	fmt.Fprint(rw, "Please refer to README.md for API usage")
}

func (s *Server) versionHandler(rw http.ResponseWriter, r *http.Request) {
	fmt.Fprint(rw, s.version)
}

func (s *Server) statusCheckHandler(rw http.ResponseWriter, r *http.Request) {
	rw.WriteHeader(200)

	if _, err := rw.Write([]byte("OK")); err != nil {
		log.Error("failed to write status check")
	}
}

/******************************************************************************
 *
 * Create the listen string
 *
 ******************************************************************************/
func (s *Server) listenAddr() string {
	return fmt.Sprintf("%s:%d", s.address, s.port)
}

/******************************************************************************
 *
 * Pass requests to the handler
 *
 ******************************************************************************/
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

/******************************************************************************
 *
 * Start the HTTP server
 *
 ******************************************************************************/
func (s *Server) Start(controlChan chan ControlMessage) error {
	listenAddr := s.listenAddr()
	log.Infof("HTTPserver: Listening on '%s'", listenAddr)

	for {
		select {
		case msg := <-controlChan:
			switch msg {
			case ControlMessageStart:
				// We're ready!
				log.Debug("HTTPserver: Control Message: Start")
				continue
			case ControlMessageQuit:
				log.Debug("HTTPserver: Control Message: Quit")
				controlChan <- ControlMessageDone // Signal exit

				return nil
			}
		default:
			if err := http.ListenAndServe(listenAddr, s); err != nil {
				log.Errorf("HTTPserver: Cannot start server: %v", err)
				return err
			}
		}
	}
}
