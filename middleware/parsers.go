package middleware

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	"github.com/gin-gonic/gin"

	"github.com/thoas/picfit/engine"
	"github.com/thoas/picfit/hash"
	"github.com/thoas/picfit/image"
	"github.com/thoas/picfit/util"
)

const sigParamName = "sig"

var parametersReg = regexp.MustCompile(`(?:(?P<sig>\w+)/)?(?P<op>\w+)/(?:(?P<w>\d+))?x(?:(?P<h>\d+))?/(?P<path>[\w\-/.]+)`)

// ParametersParser matches parameters to query string
func ParametersParser() gin.HandlerFunc {
	return func(c *gin.Context) {
		result := c.Param("parameters")

		if result != "" {
			match := parametersReg.FindStringSubmatch(result)

			parameters := make(map[string]string)

			for i, name := range parametersReg.SubexpNames() {
				if i != 0 && match[i] != "" {
					parameters[name] = match[i]
				}
			}

			c.Set("parameters", parameters)
		} else {
			if c.Query("url") == "" && c.Query("path") == "" {
				c.String(http.StatusBadRequest, "Request should contains parameters or query string")
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// KeyParser injects an unique key from query parameters
func KeyParser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var queryString map[string]string

		params, exists := c.Get("parameters")

		if exists {
			queryString = params.(map[string]string)
		} else {
			queryString = make(map[string]string)
		}

		for k, v := range c.Request.URL.Query() {
			queryString[k] = v[0]
		}

		sorted := util.SortMapString(queryString)

		delete(sorted, sigParamName)

		serialized := hash.Serialize(sorted)

		key := hash.Tokey(serialized)

		c.Set("key", key)
		c.Set("parameters", queryString)

		c.Next()
	}
}

// URLParser extracts the url query string and add a url.URL to the context
func URLParser(mimetypeDetectorType string) gin.HandlerFunc {
	mimetypeDetector := image.GetMimetypeDetector(mimetypeDetectorType)

	return func(c *gin.Context) {
		value := c.Query("url")

		if value != "" {
			url, err := url.Parse(value)

			if err != nil {
				c.String(http.StatusBadRequest, fmt.Sprintf("URL %s is not valid", value))
				c.Abort()
				return
			}

			mimetype, _ := mimetypeDetector(url)

			_, ok := image.Extensions[mimetype]

			if !ok {
				c.String(http.StatusBadRequest, fmt.Sprintf("Mimetype %s is not supported", mimetype))
				c.Abort()
				return
			}

			c.Set("url", url)
			c.Set("mimetype", mimetype)
		}

		c.Next()
	}
}

// OperationParser extracts the operation and add it to the context
func OperationParser() gin.HandlerFunc {
	return func(c *gin.Context) {
		operations := c.QueryArray("op")
		if len(operations) == 0 {
			c.String(http.StatusBadRequest, "`op` parameter or query string cannot be empty")
			c.Abort()
			return
		}

		var ops []engine.Operation
		fmt.Println(operations)
		for _, operation := range operations {
			op, ok := engine.Operations[operation]
			if !ok {
				c.String(http.StatusBadRequest, fmt.Sprintf("Invalid method %s or invalid parameters", operation))
				c.Abort()
				return
			}
			ops = append(ops, op)
		}
		c.Set("op", ops)
		c.Next()
	}
}
