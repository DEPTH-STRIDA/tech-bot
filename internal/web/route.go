package web

import (
	"net/http"

	"github.com/gorilla/mux"
)

// Маршрутизатор
func (app *WebApp) SetRoutes() *mux.Router {
	router := mux.NewRouter()

	// Ограничение количества запросов от одного IP
	router.Use(LimitMiddleware)

	// router.HandleFunc("/", app.HandleHome).Methods("GET")
	// router.HandleFunc("/replace", app.HandleReplace).Methods("GET")
	// router.HandleFunc("/login", app.HandleLogin).Methods("GET")
	router.HandleFunc("/", app.HandleReplace).Methods("GET")

	// router.HandleFunc("/notification/login", app.HandleNotificationLogin).Methods("POST")
	// router.HandleFunc("/notification/validation", app.HandleNotificationValidation).Methods("POST")
	// router.HandleFunc("/notification/logout", app.HandleNotificationLogOut).Methods("POST")

	router.HandleFunc("/getData", app.HandleGetSelectData).Methods("GET")
	router.HandleFunc("/getHistoryData", app.HandleGetHistoryData).Methods("GET")
	//router.HandleFunc("/getTimeZone", app.HandleGetTimeZone).Methods("GET")

	router.HandleFunc("/postSetData", app.HandlePostSetData).Methods("POST")
	router.HandleFunc("/postEditData", app.HandlePostEditData).Methods("POST")
	router.HandleFunc("/postDeleteData", app.HandlePostDeleteData).Methods("POST")

	/////////////////////////////////////////////////////////////////////////////////////////
	//////////////////                   ADMIN                    ///////////////////////////
	/////////////////////////////////////////////////////////////////////////////////////////
	router.HandleFunc("/internal/admin", app.HandleInternalAdmin).Methods("GET")
	router.HandleFunc("/internal/admin-menu", app.HandleInternalAdminMenu).Methods("GET")

	staticDir := "../../ui/tech_bot/static/"
	fileServer := http.FileServer(http.Dir(staticDir))
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fileServer))

	return router
}
