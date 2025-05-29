package graph

import (
	"fmt"
	"net/http"

	dtdv0 "github.com/emereshub/twinscale-operator/api/dtd/v0"
)

func NewTwinGraphServer() TwinGraphServer {
	return &twinGraphServer{
		twinGraphInstance: NewEmptyTwinInstanceGraph(),
	}
}

type TwinGraphServer interface {
	UpdateGraphFunc(twinInstances []dtdv0.TwinInstance)
	HandleGraphFunc() http.HandlerFunc
}

type twinGraphServer struct {
	twinGraphInstance TwinInstanceGraph
}

func (t *twinGraphServer) HandleGraphFunc() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonFormat, _ := t.twinGraphInstance.MarshalJson()
		w.WriteHeader(200)
		w.Write(jsonFormat)
	})
}

func (t *twinGraphServer) UpdateGraphFunc(twinInstances []dtdv0.TwinInstance) {
	twinGraphInstance := NewEmptyTwinInstanceGraph()

	for _, twinInstance := range twinInstances {
		twinGraphInstance.AddVertex(twinInstance)
	}

	for _, twinInstance := range twinInstances {
		for _, relationship := range twinInstance.Spec.TwinInstanceRelationships {
			twinInstanceVertex := twinGraphInstance.GetVertex(relationship.Instance)
			if twinInstanceVertex != nil {
				err := twinGraphInstance.AddEdge(twinInstance, *twinInstanceVertex)
				if err != nil {
					fmt.Println(err)
				}
			}
		}
	}

	t.twinGraphInstance = twinGraphInstance
}
