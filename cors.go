package butler

import "github.com/labstack/echo/v4/middleware"

type CorsSettings struct {
	config middleware.CORSConfig
}

// MaxAge determines the value of the Access-Control-Max-Age response header.
// This header indicates how long (in seconds) the results of a preflight
// request can be cached.
// The header is set only if MaxAge != 0, negative value sends "0" which instructs browsers not to cache that response.
//
// Optional. Default value 0 - meaning header is not sent.
//
// See also: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Max-Age
func (s *CorsSettings) SetMaxAge(maxAge int) {
	s.config.MaxAge = maxAge
}

// ExposeHeaders determines the value of Access-Control-Expose-Headers, which
// defines a list of headers that clients are allowed to access.
//
// Optional. Default value []string{}, in which case the header is not set.
//
// See also: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Expose-Header
func (s *CorsSettings) ExposeHeaders(headers ...string) {
	s.config.ExposeHeaders = append(s.config.ExposeHeaders, headers...)
}

// AllowOrigins determines the value of the Access-Control-Allow-Origin
// response header.  This header defines a list of origins that may access the
// resource.  The wildcard characters '*' and '?' are supported and are
// converted to regex fragments '.*' and '.' accordingly.
//
// Security: use extreme caution when handling the origin, and carefully
// validate any logic. Remember that attackers may register hostile domain names.
// See https://blog.portswigger.net/2016/10/exploiting-cors-misconfigurations-for.html
//
// Optional. Default value []string{"*"}.
//
// See also: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Origin
func (s *CorsSettings) AllowOrigin(origins ...string) {
	s.config.AllowOrigins = append(s.config.AllowOrigins, origins...)
}

// AllowHeaders determines the value of the Access-Control-Allow-Headers
// response header.  This header is used in response to a preflight request to
// indicate which HTTP headers can be used when making the actual request.
//
// Optional. Default value []string{}.
//
// See also: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Headers
func (s *CorsSettings) AllowHeaders(headers ...string) {
	s.config.AllowHeaders = append(s.config.AllowHeaders, headers...)
}

// AllowMethods determines the value of the Access-Control-Allow-Methods
// response header.  This header specified the list of methods allowed when
// accessing the resource.  This is used in response to a preflight request.
//
// Optional. Default value DefaultCORSConfig.AllowMethods.
// If `allowMethods` is left empty, this middleware will fill for preflight
// request `Access-Control-Allow-Methods` header value
// from `Allow` header that echo.Router set into context.
//
// See also: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Methods
func (s *CorsSettings) AllowMethods(methods ...string) {
	s.config.AllowMethods = append(s.config.AllowMethods, methods...)
}

// AllowCredentials determines the value of the
// Access-Control-Allow-Credentials response header.  This header indicates
// whether or not the response to the request can be exposed when the
// credentials mode (Request.credentials) is true. When used as part of a
// response to a preflight request, this indicates whether or not the actual
// request can be made using credentials.  See also
// [MDN: Access-Control-Allow-Credentials].
//
// Optional. Default value false, in which case the header is not set.
//
// Security: avoid using `AllowCredentials = true` with `AllowOrigins = *`.
// See "Exploiting CORS misconfigurations for Bitcoins and bounties",
// https://blog.portswigger.net/2016/10/exploiting-cors-misconfigurations-for.html
//
// See also: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Credentials
func (s *CorsSettings) AllowCredentials(allow bool) {
	s.config.AllowCredentials = allow
}
