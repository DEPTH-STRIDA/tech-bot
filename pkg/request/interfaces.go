package request

// Request представляет функцию, которая будет выполнена с возможным возвратом ошибки.
type Request func() error
