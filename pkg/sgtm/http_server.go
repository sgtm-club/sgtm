package sgtm

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime/debug"
	"strings"

	"github.com/cespare/hutil/apachelog"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/jsonp"
	"github.com/go-chi/render"
	packr "github.com/gobuffalo/packr/v2"
	"github.com/gogo/gateway"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/ikeikeikeike/go-sitemap-generator/v2/stm"
	"github.com/oklog/run"
	"github.com/rs/cors"
	"github.com/soheilhy/cmux"
	chilogger "github.com/treastech/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gopkg.in/natefinch/lumberjack.v2"
	"moul.io/banner"
	"moul.io/sgtm/pkg/sgtmpb"
)

type serverDriver struct {
	listener net.Listener
}

func (svc *Service) StartServer() error {
	fmt.Fprintln(os.Stderr, banner.Inline("server"))
	svc.logger.Debug("starting server", zap.String("bind", svc.opts.ServerBind))

	// listeners
	var err error
	svc.server.listener, err = net.Listen("tcp", svc.opts.ServerBind)
	if err != nil {
		return err
	}
	smux := cmux.New(svc.server.listener)
	smux.HandleError(func(err error) bool {
		svc.logger.Warn("cmux error", zap.Error(err))
		return true
	})
	grpcListener := smux.MatchWithWriters(cmux.HTTP2MatchHeaderFieldPrefixSendSettings("content-type", "application/grpc"))
	httpListener := smux.Match(cmux.HTTP2(), cmux.HTTP1())

	// grpc server
	grpcServer := svc.grpcServer()
	var gr run.Group
	gr.Add(func() error {
		err := grpcServer.Serve(grpcListener)
		if err != cmux.ErrListenerClosed {
			return err
		}
		return nil
	}, func(error) {
		err := grpcListener.Close()
		if err != nil {
			svc.logger.Warn("close gRPC listener", zap.Error(err))
		}
	})

	// http server
	httpServer, err := svc.httpServer()
	if err != nil {
		return err
	}
	gr.Add(func() error {
		err := httpServer.Serve(httpListener)
		if err != cmux.ErrListenerClosed {
			return err
		}
		return nil
	}, func(error) {
		ctx, cancel := context.WithTimeout(svc.ctx, svc.opts.ServerShutdownTimeout)
		err := httpServer.Shutdown(ctx)
		if err != nil {
			svc.logger.Warn("shutdown HTTP server", zap.Error(err))
		}
		defer cancel()
		err = httpListener.Close()
		if err != nil {
			svc.logger.Warn("close HTTP listener", zap.Error(err))
		}
	})

	// cmux
	gr.Add(
		smux.Serve,
		func(err error) {
			if err != nil {
				svc.logger.Warn("cmux terminated", zap.Error(err))
			} else {
				svc.logger.Debug("cmux terminated")
			}
		},
	)

	// context
	gr.Add(func() error {
		<-svc.ctx.Done()
		svc.logger.Debug("parent ctx done")
		return nil
	}, func(error) {})

	return gr.Run()
}

func (svc *Service) CloseServer(err error) {
	svc.logger.Debug("closing server", zap.Bool("was-started", svc.server.listener != nil), zap.Error(err))
	if svc.server.listener != nil {
		svc.server.listener.Close()
	}
	svc.cancel()
}

func (svc *Service) ServerListenerAddr() string {
	return svc.server.listener.Addr().String()
}

func (svc *Service) httpServer() (*http.Server, error) {
	r := chi.NewRouter()
	r.Use(cors.New(cors.Options{
		AllowedOrigins:   strings.Split(svc.opts.ServerCORSAllowedOrigins, ","),
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}).Handler)
	r.Use(chilogger.Logger(svc.logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(svc.opts.ServerRequestTimeout))
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	gatewayMetadataAnnotator := func(_ context.Context, r *http.Request) metadata.MD {
		mdmap := make(map[string]string)
		cookieToken, err := r.Cookie("oauth-token")
		if err == nil {
			mdmap["oauth-token"] = cookieToken.Value
		}
		return metadata.New(mdmap)
	}
	gatewayCustomMatcher := func(key string) (string, bool) {
		if key == "Cookie" {
			return key, true
		}
		return runtime.DefaultHeaderMatcher(key)
	}
	runtimeMux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &gateway.JSONPb{
			EmitDefaults: false,
			Indent:       "  ",
			OrigName:     true,
		}),
		runtime.WithProtoErrorHandler(runtime.DefaultHTTPProtoErrorHandler),
		runtime.WithMetadata(gatewayMetadataAnnotator),
		runtime.WithIncomingHeaderMatcher(gatewayCustomMatcher),
	)
	var gwmux http.Handler = runtimeMux
	dialOpts := []grpc.DialOption{grpc.WithInsecure()}

	err := sgtmpb.RegisterWebAPIHandlerFromEndpoint(svc.ctx, runtimeMux, svc.ServerListenerAddr(), dialOpts)
	if err != nil {
		return nil, err
	}

	var handler http.Handler = gwmux

	// api
	r.Route("/api", func(r chi.Router) {
		//r.Use(auth(opts.BasicAuth, opts.Realm, opts.AuthSalt))
		r.Use(jsonp.Handler)
		r.Mount("/", handler)
	})

	// fs
	staticBox := packr.New("static", "../../static")
	srcBox := packr.New("src", ".")
	if !strings.HasSuffix(srcBox.ResolutionDir, "pkg/sgtm") { // strange bug -> no problem for production, but issue in dev if this line doesn't exist
		srcBox.ResolutionDir += "/pkg/sgtm"
	}

	// dynamic pages
	error404Page := svc.error404Page(srcBox)
	{
		svc.errRenderHTML = svc.errorPage(srcBox)
		r.Get("/", svc.homePage(srcBox))
		r.Get("/settings", svc.settingsPage(srcBox))
		r.Post("/settings", svc.settingsPage(srcBox))
		r.Get("/@{user_slug}", svc.profilePage(srcBox))
		r.Get("/open", svc.openPage(srcBox))
		r.Get("/new", svc.newPage(srcBox))
		r.Post("/new", svc.newPage(srcBox))
		r.Get("/post/{post_slug}", svc.postPage(srcBox))
		r.Get("/post/{post_slug}/edit", svc.postEditPage(srcBox))
		r.Post("/post/{post_slug}/edit", svc.postEditPage(srcBox))
		r.Get("/post/{post_slug}/sync", svc.postSyncPage(srcBox))
		// FIXME: r.Use(ModeratorOnly) + r.Get("/moderator")
		// FIXME: r.Use(AdminOnly) + r.Get("/admin")
		r.Get("/rss.xml", svc.rssPage(srcBox))
	}

	// special pages
	{
		sm := svc.generateSitemap()
		r.Get("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(sm.XMLContent())
		})
	}

	// auth
	{
		r.Get("/login", svc.httpAuthLogin)
		r.Get("/logout", svc.httpAuthLogout)
		r.Get("/auth/callback", svc.httpAuthCallback)
	}

	// static files & 404
	{
		fs := http.FileServer(staticBox)
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, ".tmpl.html") { // hide sensitive files
				error404Page(w, r)
				return
			}
			if r.URL.Path != "/" {
				_, err := staticBox.FindString(r.URL.Path)
				if err != nil {
					error404Page(w, r)
					return
				}
			}
			fs.ServeHTTP(w, r)
		})
	}

	// pprof
	if svc.opts.ServerWithPprof {
		r.HandleFunc("/debug/pprof/*", pprof.Index)
		r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		r.HandleFunc("/debug/pprof/profile", pprof.Profile)
		r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		r.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	// configure server
	http.DefaultServeMux = http.NewServeMux() // disables default handlers registered by importing net/http/pprof for security reasons

	//apacheLogFormat := `%h - %u %{02/Jan/2006 15:04:05 -0700}t "%m %U%q %H" %s %b %D`
	apacheLogFormat := `%h - %{Sgtm-User-Slug}o %t "%r" %s %b "%{Referer}i" "%{User-Agent}i"`
	ljack := lumberjack.Logger{
		Filename:   "./logs/access.log",
		MaxSize:    500, // megabytes
		MaxBackups: 10,
		MaxAge:     365,
		Compress:   true,
	}
	loggedHandler := apachelog.NewHandler(apacheLogFormat, r, &ljack)

	return &http.Server{
		Addr:    "osef",
		Handler: loggedHandler,
	}, nil
}

func (svc *Service) generateSitemap() *stm.Sitemap {
	sm := stm.NewSitemap(1)
	sm.SetDefaultHost("https://sgtm.club")
	sm.SetVerbose(true)
	sm.Create()
	// special pages
	sm.Add(stm.URL{
		{"loc", "/"},
		{"changefreq", "daily"},
		{"priority", 1},
	})
	sm.Add(stm.URL{
		{"loc", "/open"},
		{"changefreq", "daily"},
	})
	// users
	{
		var users []*sgtmpb.User
		err := svc.rodb.
			Find(&users).
			Error
		if err != nil {
			svc.logger.Error("query users for sitemap", zap.Error(err))
		} else {
			for _, user := range users {
				sm.Add(stm.URL{
					{"loc", user.CanonicalURL()},
					{"changefreq", "weekly"},
				})
			}
		}
	}
	// posts
	{
		var posts []*sgtmpb.Post
		err := svc.rodb.
			Where(sgtmpb.Post{
				Visibility: sgtmpb.Visibility_Public,
				Kind:       sgtmpb.Post_TrackKind,
			}).
			Find(&posts).
			Error
		if err != nil {
			svc.logger.Error("query posts for sitemap", zap.Error(err))
		} else {
			for _, post := range posts {
				sm.Add(stm.URL{
					{"loc", post.CanonicalURL()},
					{"changefreq", "weekly"},
				})
			}
		}
	}
	return sm
}

func (svc *Service) AuthFuncOverride(ctx context.Context, path string) (context.Context, error) {
	// accept everything
	return ctx, nil
}

func (svc *Service) grpcServer() *grpc.Server {
	authFunc := func(context.Context) (context.Context, error) {
		return nil, errors.New("auth: dummy function, see AuthFuncOverride")
	}
	recoveryOpts := []grpc_recovery.Option{}
	if svc.logger.Check(zap.DebugLevel, "") != nil {
		recoveryOpts = append(recoveryOpts, grpc_recovery.WithRecoveryHandlerContext(func(ctx context.Context, p interface{}) error {
			log.Println("stacktrace from panic: \n" + string(debug.Stack()))
			return status.Errorf(codes.Internal, "recover: %s", p)
		}))
	}
	serverStreamOpts := []grpc.StreamServerInterceptor{grpc_recovery.StreamServerInterceptor(recoveryOpts...)}
	serverUnaryOpts := []grpc.UnaryServerInterceptor{grpc_recovery.UnaryServerInterceptor(recoveryOpts...)}
	serverStreamOpts = append(serverStreamOpts,
		grpc_auth.StreamServerInterceptor(authFunc),
		//grpc_ctxtags.StreamServerInterceptor(),
		grpc_zap.StreamServerInterceptor(svc.logger),
	)
	serverUnaryOpts = append(
		serverUnaryOpts,
		grpc_auth.UnaryServerInterceptor(authFunc),
		//grpc_ctxtags.UnaryServerInterceptor(),
		grpc_zap.UnaryServerInterceptor(svc.logger),
	)
	if svc.logger.Check(zap.DebugLevel, "") != nil {
		serverStreamOpts = append(serverStreamOpts, grpcServerStreamInterceptor())
		serverUnaryOpts = append(serverUnaryOpts, grpcServerUnaryInterceptor())
	}
	serverStreamOpts = append(serverStreamOpts, grpc_recovery.StreamServerInterceptor(recoveryOpts...))
	serverUnaryOpts = append(serverUnaryOpts, grpc_recovery.UnaryServerInterceptor(recoveryOpts...))
	grpcServer := grpc.NewServer(
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(serverStreamOpts...)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(serverUnaryOpts...)),
	)
	sgtmpb.RegisterWebAPIServer(grpcServer, svc)

	return grpcServer
}

func grpcServerStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		err := handler(srv, stream)
		if err != nil {
			log.Printf("%+v", err)
		}
		return err
	}
}

func grpcServerUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ret, err := handler(ctx, req)
		if err != nil {
			log.Printf("%+v", err)
		}
		return ret, err
	}
}
