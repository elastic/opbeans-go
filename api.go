package main

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/cache"
	"github.com/gin-contrib/cache/persistence"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func addAPIHandlers(r *gin.RouterGroup, db *sql.DB, logger *logrus.Logger) {
	h := apiHandlers{db, logger}
	r.GET("/stats", h.getStats)
	r.GET("/products", h.getProducts)
	r.GET("/products/:id", h.getProductDetails)
	r.GET("/products/:id/customers", h.getProductCustomers)
	r.GET("/types", h.getProductTypes)
	r.GET("/customers", h.getCustomers)
	r.GET("/customers/:id", h.getCustomerDetails)
	r.GET("/orders", h.getOrders)
	r.GET("/orders/:id", h.getOrderDetails)
}

type apiHandlers struct {
	db  *sql.DB
	log *logrus.Logger
}

func (h apiHandlers) getStats(c *gin.Context) {
	cacheValue, _ := c.Get(cache.CACHE_MIDDLEWARE_KEY)
	cache := *cacheValue.(*persistence.CacheStore)

	// TODO(axw) tag transaction with "served_from_cache"

	const cacheKey = "shop-stats"
	var stats *Stats
	err := cache.Get(cacheKey, &stats)
	switch err {
	case nil:
		h.log.Debug("serving stats from cache")
		c.JSON(http.StatusOK, stats)
		return
	case persistence.ErrCacheMiss:
		// fetch and cache below
		break
	default:
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	stats, err = getStats(h.db)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if err := cache.Set(cacheKey, stats, time.Minute); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	h.log.Debug("cached stats")
	c.JSON(http.StatusOK, stats)
}

func (h apiHandlers) getProducts(c *gin.Context) {
	products, err := getProducts(h.db)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, products)
}

func (h apiHandlers) getTopProducts(c *gin.Context) {
	products, err := getTopProducts(h.db)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, products)
}

func (h apiHandlers) getProductDetails(c *gin.Context) {
	idString := c.Param("id")
	if idString == "top" {
		products, err := getTopProducts(h.db)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, products)
		return
	}

	// Product by ID.
	id, err := strconv.Atoi(idString)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	product, err := getProduct(h.db, id)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if product == nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.JSON(http.StatusOK, product)
}

func (h apiHandlers) getProductCustomers(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	limit := 1000
	if countString := c.Param("count"); countString != "" {
		limit, err = strconv.Atoi(countString)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}
	customers, err := getProductCustomers(h.db, id, limit)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, customers)
}

func (h apiHandlers) getProductTypes(c *gin.Context) {
	productTypes, err := getProductTypes(h.db)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, productTypes)
}

func (h apiHandlers) getCustomers(c *gin.Context) {
	customers, err := getCustomers(h.db)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, customers)
}

func (h apiHandlers) getCustomerDetails(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	customer, err := getCustomer(h.db, id)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if customer == nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.JSON(http.StatusOK, customer)
}

func (h apiHandlers) getOrders(c *gin.Context) {
	orders, err := getOrders(h.db)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, orders)
}

func (h apiHandlers) getOrderDetails(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	customer, err := getOrder(h.db, id)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if customer == nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.JSON(http.StatusOK, customer)
}
