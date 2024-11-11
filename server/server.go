package server

import (
	"context"
	"net/http"
	"rms/handler"
	"rms/middlewares"
	"rms/models"
	"time"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	chi.Router
	server *http.Server
}

const (
	readTimeout       = 5 * time.Minute
	readHeaderTimeout = 30 * time.Second
	writeTimeout      = 5 * time.Minute
)

// SetupRoutes provides all the routes that can be used
func SetupRoutes() *Server {
	router := chi.NewRouter()
	router.Route("/v1", func(v1 chi.Router) {
		v1.Use(middlewares.CommonMiddlewares()...)
		v1.Route("/", func(public chi.Router) {
			public.Post("/login", handler.LoginUser)
			public.Route("/", func(authRouts chi.Router) {
				authRouts.Use(middlewares.AuthMiddleware)
				authRouts.Get("/", handler.GetInfo)
				authRouts.Put("/", handler.UpdateSelfInfo)
				authRouts.Delete("/logout", handler.Logout)
				authRouts.Get("/restaurants", handler.GetRestaurants)
				authRouts.Get("/restaurant/{restaurantId}/dishes", handler.GetRestaurantsDishes)
				authRouts.Route("/user", func(user chi.Router) {
					user.Use(middlewares.ShouldHaveRole([]models.Role{models.RoleUser}))
					user.Group(userRoutes)
				})
				authRouts.Route("/sub-admin", func(subAdmin chi.Router) {
					subAdmin.Use(middlewares.ShouldHaveRole([]models.Role{models.RoleAdmin, models.RoleSubAdmin}))
					subAdmin.Group(subAdminRoutes)
				})
				authRouts.Route("/admin", func(admin chi.Router) {
					admin.Use(middlewares.ShouldHaveRole([]models.Role{models.RoleAdmin}))
					admin.Group(adminRoutes)
				})
			})
		})
	})
	return &Server{
		Router: router,
	}
}

func (svc *Server) Run(port string) error {
	svc.server = &http.Server{
		Addr:              port,
		Handler:           svc.Router,
		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
	}
	return svc.server.ListenAndServe()
}

func (svc *Server) Shutdown(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return svc.server.Shutdown(ctx)
}
