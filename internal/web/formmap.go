package web

import (
	"easycodeapp/pkg/request"
	"sync"
)

type FormMap struct {
	sync.Mutex
	c map[string]request.RequestHandler
}

func NewFormMap() (*FormMap, error) {
	var c map[string]request.RequestHandler
	return &FormMap{c: c}, nil
}

func (app *FormMap) HandleRequest() {

}
