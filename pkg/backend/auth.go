package backend

import (
	"context"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	auth "k8s.io/api/authentication/v1"
	auth2 "k8s.io/api/authorization/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//
// Default auth provider.
var DefaultAuth = Auth{
	TTL: time.Second * 10,
}

//
// Authorized by k8s bearer token SAR.
// Token must have "*" on the cluster, like cluster-admin
type Auth struct {
	// k8s API writer.
	Writer client.Writer
	// Cached token TTL.
	TTL time.Duration
	// Mutex.
	mutex sync.Mutex
	// Token cache.
	cache map[string]time.Time
}

//
// Authenticate token.
func (r *Auth) Permit(ctx *gin.Context) {
	p := "must-gather"
	r.mutex.Lock()
	defer r.mutex.Unlock()
	status := http.StatusOK
	if r.cache == nil {
		r.cache = make(map[string]time.Time)
	}
	r.prune()
	token := r.token(ctx)
	if token == "" {
		log.Println("Authorization check - missing bearer token.")
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	key := r.key(token, p)
	if t, found := r.cache[key]; found {
		if time.Since(t) <= r.TTL {
			log.Println("Authorization check - allowed from cache.")
			return
		}
	}
	allowed, err := r.permitClusterAdmin(token)
	if err != nil {
		log.Println(err, "Authorization check - token auth failed.")
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	if allowed {
		r.cache[key] = time.Now()
		log.Printf("Authorization check - allowed")
		return
	} else {
		status = http.StatusForbidden
		delete(r.cache, token)
		log.Println(
			http.StatusText(status),
			"token",
			token)
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
}

//
// Authenticate token for a custer admin which is required by oc adm must-gather
func (r *Auth) permitClusterAdmin(token string) (allowed bool, err error) {
	var cfg *rest.Config

	tr := &auth.TokenReview{
		Spec: auth.TokenReviewSpec{
			Token: token,
		},
	}

	if os.Getenv("USE_KUBECONFIG") == "true" {
		// Get cluster API endpoint for local execution
		// using kubeconfig file
		kubeconfig := os.Getenv("KUBECONFIG")
		if len(kubeconfig) == 0 {
			kubeconfig = filepath.Join(homedir.HomeDir(), ".kube", "config")
		}

		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	} else {
		// Get in-cluster config from the pod environment to find cluster API host/port
		cfg, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	}

	// Reset bearer token to the one captured from API request
	cfg.BearerTokenFile = ""
	cfg.BearerToken = token

	// Call cluster API
	w, err := r.writer(cfg)
	if err != nil {
		log.Printf("Error: cluster API writer %v", w)
		log.Println(err)
		return
	}
	err = w.Create(context.TODO(), tr)
	if err != nil {
		log.Printf("Error: create token review %v", tr)
		log.Println(err)
		return
	}
	if !tr.Status.Authenticated {
		log.Printf("Failed token auth: %v", tr.Status)
		return
	}
	user := tr.Status.User
	extra := map[string]auth2.ExtraValue{}
	for k, v := range user.Extra {
		extra[k] = append(
			auth2.ExtraValue{},
			v...)
	}
	ar := &auth2.SubjectAccessReview{
		Spec: auth2.SubjectAccessReviewSpec{
			ResourceAttributes: &auth2.ResourceAttributes{
				Group:     "*",
				Resource:  "*",
				Namespace: "*",
				Name:      "",
				Verb:      "*",
			},
			Extra:  extra,
			Groups: user.Groups,
			User:   user.Username,
			UID:    user.UID,
		},
	}
	err = w.Create(context.TODO(), ar)
	if err != nil {
		log.Printf("Error: create SubjectAccessReview %v", ar)
		log.Println(err)
		return
	}

	allowed = ar.Status.Allowed
	return
}

//
// Extract token.
func (r *Auth) token(ctx *gin.Context) (token string) {
	header := ctx.GetHeader("Authorization")
	fields := strings.Fields(header)
	if len(fields) == 2 && fields[0] == "Bearer" {
		token = fields[1]
	}

	return
}

//
// Prune the cache.
// Evacuate expired tokens.
func (r *Auth) prune() {
	for token, t := range r.cache {
		if time.Since(t) > r.TTL {
			delete(r.cache, token)
		}
	}
}

//
// Cache key.
func (r *Auth) key(token, p string) string {
	return path.Join(
		token,
		p)
}

//
// Build API writer.
func (r *Auth) writer(cfg *rest.Config) (w client.Writer, err error) {
	w, err = client.New(
		cfg,
		client.Options{
			Scheme: scheme.Scheme,
		})
	if err == nil {
		r.Writer = w
	}

	return
}
