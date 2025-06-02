package httpbutler

import (
	"fmt"
	"time"
)

type HttpCachePolicy struct {
	// If it's detected that the response ETag matches the ETag in request If-None-Match header, it will only send
	// a 304 (NotModified) response without content. This behavior can be disabled by setting this option to true.
	DisableAutoResponse bool
	// ETags area automatically generated for all responses with non-nil HttpCachePolicy. This behavior can be disabled
	// by setting this option to true.
	DisableETagGeneration bool
	// The max-age=N response directive indicates that the response remains fresh until N seconds after the response is
	// generated.
	MaxAge time.Duration
	// The s-maxage response directive indicates how long the response remains fresh in a shared cache. The s-maxage
	// directive is ignored by private caches, and overrides the value specified by the max-age directive or the
	// Expires header for shared caches, if they are present.
	SMaxAge time.Duration
	// The stale-while-revalidate response directive indicates that the cache could reuse a stale response while it
	// revalidates it to a cache.
	StaleWhileRevalidate time.Duration
	// The stale-if-error response directive indicates that the cache can reuse a stale response when an upstream
	// server generates an error, or when the error is generated locally. Here, an error is considered any response
	// with a status code of 500, 502, 503, or 504.
	StaleIfError time.Duration
	// The immutable response directive indicates that the response will not be updated while it's fresh.
	Immutable bool
	// The no-store response directive indicates that any caches of any kind (private or shared) should not store this
	// response.
	NoStore bool
	// The no-cache response directive indicates that the response can be stored in caches, but the response must be
	// validated with the origin server before each reuse, even when the cache is disconnected from the origin server.
	NoCache bool
	// The must-revalidate response directive indicates that the response can be stored in caches and can be reused
	// while fresh. If the response becomes stale, it must be validated with the origin server before reuse.
	MustRevalidate bool
	// he private response directive indicates that the response can be stored only in a private cache (e.g.,
	// local caches in browsers).
	//
	// If set to false, or not set - public will be assumed.
	Private bool
	// The proxy-revalidate response directive is the equivalent of must-revalidate, but specifically for shared
	// caches only.
	ProxyRevalidate bool
	//The must-understand response directive indicates that a cache should store the response only if it understands
	// the requirements for caching based on status code.
	//
	// must-understand should be coupled with no-store for fallback behavior.
	MustUnderstand bool
	// Some intermediaries transform content for various reasons. For example, some convert images to reduce transfer
	// size. In some cases, this is undesirable for the content provider.
	//
	// no-transform indicates that any intermediary (regardless of whether it implements a cache) shouldn't transform
	// the response contents.
	NoTransform bool
}

func (policy HttpCachePolicy) ToString() string {
	value := ""
	if policy.Private {
		value = "private"
	} else {
		value = "public"
	}

	if policy.MaxAge > 0 {
		value = fmt.Sprintf("%s, max-age=%v", value, int64(policy.MaxAge.Seconds()))
	}
	if policy.SMaxAge > 0 {
		value = fmt.Sprintf("%s, s-maxage=%v", value, int64(policy.SMaxAge.Seconds()))
	}
	if policy.NoStore {
		value = fmt.Sprintf("%s, no-store", value)
	}
	if policy.NoCache {
		value = fmt.Sprintf("%s, no-cache", value)
	}
	if policy.Immutable {
		value = fmt.Sprintf("%s, immutable", value)
	}
	if policy.MustRevalidate {
		value = fmt.Sprintf("%s, must-revalidate", value)
	}
	if policy.ProxyRevalidate {
		value = fmt.Sprintf("%s, proxy-revalidate", value)
	}
	if policy.MustUnderstand {
		value = fmt.Sprintf("%s, must-understand", value)
	}
	if policy.NoTransform {
		value = fmt.Sprintf("%s, no-transform", value)
	}
	if policy.StaleWhileRevalidate > 0 {
		value = fmt.Sprintf("%s, stale-while-revalidate=%v", value, int64(policy.StaleWhileRevalidate.Seconds()))
	}
	if policy.StaleIfError > 0 {
		value = fmt.Sprintf("%s, stale-if-error=%v", value, int64(policy.StaleIfError.Seconds()))
	}
	return value
}
