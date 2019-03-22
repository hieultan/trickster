/**
* Copyright 2018 Comcast Cable Communications Management, LLC
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
* http://www.apache.org/licenses/LICENSE-2.0
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package proxy

import (
	"net/http"
	"strconv"

	"github.com/Comcast/trickster/internal/cache"
	"github.com/Comcast/trickster/internal/util/log"
	"github.com/Comcast/trickster/internal/util/metrics"
	"github.com/Comcast/trickster/pkg/locks"
)

// ObjectProxyCacheRequest provides a Basic HTTP Reverse Proxy/Cache
func ObjectProxyCacheRequest(r *Request, w http.ResponseWriter, client Client, cache cache.Cache, ttl int, refresh bool, noLock bool) {

	key := client.DeriveCacheKey(r.URL.Path, r.URL.Query(), "", "")

	if !noLock {
		locks.Acquire(key)
		defer locks.Release(key)
	}

	if !refresh {
		if d, err := QueryCache(cache, key); err == nil {
			metrics.ProxyRequestStatus.WithLabelValues(r.OriginName, r.OriginType, r.HTTPMethod, crHit, "200", r.URL.Path).Inc()
			log.Debug("cache hit", log.Pairs{"key": key})
			Respond(w, d.StatusCode, d.Headers, d.Body)
			return
		}
	}

	body, resp, elapsed := Fetch(r)
	metrics.ProxyRequestStatus.WithLabelValues(r.OriginName, r.OriginType, r.HTTPMethod, crKeyMiss, strconv.Itoa(resp.StatusCode), r.URL.Path).Inc()
	metrics.ProxyRequestDuration.WithLabelValues(r.OriginName, r.OriginType, r.HTTPMethod, crKeyMiss, strconv.Itoa(resp.StatusCode), r.URL.Path).Observe(elapsed.Seconds())

	if resp.StatusCode == http.StatusOK && len(body) > 0 {
		WriteCache(cache, key, DocumentFromHTTPResponse(resp, body), ttl)
	}

	Respond(w, resp.StatusCode, resp.Header, body)
}