package main

import (
	"context"
	"fmt"
	"github.com/emicklei/go-restful"
	promapi "github.com/prometheus/client_golang/api"
	promapiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"log"
	"net/http"
	"os"
	"time"
)

type MyHandler struct {
	GoRestfulContainer *restful.Container
}

func NewMyHandler() *MyHandler {
	gorestfulContainer := restful.NewContainer()
	gorestfulContainer.ServeMux = http.NewServeMux()
	gorestfulContainer.Router(restful.CurlyRouter{})

	return &MyHandler{
		GoRestfulContainer: gorestfulContainer,
	}
}

func (h *MyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h.GoRestfulContainer.Dispatch(w, req)
	return
}

func NewWebService(group string, version string) *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/apis/" + group + "/" + version)
	ws.Doc("API at /apis/apps/v1")
	ws.Consumes(restful.MIME_XML, restful.MIME_JSON)
	ws.Produces(restful.MIME_JSON, restful.MIME_XML) // you can specify this per route as well
	return ws
}

func registerHandler(resource string, ws *restful.WebService) {
	routes := []*restful.RouteBuilder{}

	nameParam := ws.PathParameter("name", "name of the resource").DataType("string")
	namespaceParam := ws.PathParameter("namespace", "object name and auth scope, such as for teams and projects").DataType("string")

	route := ws.GET("namespaces"+"/{namespace}/"+resource+"/{name}"+"/{metrics}").
		Produces(restful.MIME_JSON).
		Writes(Foo{}).
		To(metricsHandler).
		Param(namespaceParam).
		Param(nameParam).
		Doc("Get a KubeVirt API resources").
		Returns(http.StatusOK, "OK", Foo{}).
		Returns(http.StatusNotFound, "Not Found", "")

	routes = append(routes, route)

	for _, route := range routes {
		ws.Route(route)
	}
}

type Foo struct {
	Namespace string
	Name      string
}

func metricsHandler(req *restful.Request, res *restful.Response) {
	namespace := req.PathParameter("namespace")
	name := req.PathParameter("name")
	metrics := req.PathParameter("metrics")

	ip := os.Getenv("IP")
	port := os.Getenv("PORT")
	endpoint := fmt.Sprintf("http://%s:%s", ip, port)

	fmt.Println(endpoint)

	cfg := promapi.Config{
		Address: endpoint,
	}

	client, err := promapi.NewClient(cfg)
	if err != nil {
		fmt.Errorf("%v", err)
	}
	api := promapiv1.NewAPI(client)

	ctx := context.Background()
	r := promapiv1.Range{
		Start: time.Now().Add(-time.Minute * 10),
		End:   time.Now(),
		Step:  time.Second,
	}
	result, warnings, err := api.QueryRange(ctx, metrics+`{name="`+name+`",namespace="`+namespace+`"}`, r)
	if err != nil {
		fmt.Errorf("%v", err)
	}

	if len(warnings) > 0 {
		fmt.Errorf("%v", err)
	}

	fmt.Printf("Result:\n%v\n", result)

	res.WriteAsJson(result)
}

func main() {
	handler := NewMyHandler()

	ws := NewWebService("subresources.harvester.io", "v1")
	registerHandler("virtualmachineinstances", ws)

	handler.GoRestfulContainer.Add(ws)

	s := &http.Server{
		Addr:           ":8080",
		Handler:        handler,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())
}
