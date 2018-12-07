package main

import (
	"encoding/csv"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/cache"
	"github.com/gin-contrib/cache/persistence"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

	"go.elastic.co/apm"
)

func addAPIHandlers(r *gin.RouterGroup, db *sqlx.DB) {
	h := apiHandlers{db}
	r.GET("/stats", h.getStats)
	r.GET("/products", h.getProducts)
	r.GET("/products/:id", h.getProductDetails)
	r.GET("/products/:id/customers", h.getProductCustomers)
	r.GET("/types", h.getProductTypes)
	r.GET("/types/:id", h.getProductTypeDetails)
	r.GET("/customers", h.getCustomers)
	r.GET("/customers/:id", h.getCustomerDetails)
	r.GET("/orders", h.getOrders)
	r.GET("/orders/:id", h.getOrderDetails)
	r.POST("/orders", h.postOrder)
	r.POST("/orders/csv", h.postOrderCSV)
}

type apiHandlers struct {
	db *sqlx.DB
}

func (h apiHandlers) getStats(c *gin.Context) {
	cacheValue, _ := c.Get(cache.CACHE_MIDDLEWARE_KEY)
	cache := *cacheValue.(*persistence.CacheStore)

	const cacheKey = "shop-stats"
	var stats *Stats
	err := cache.Get(cacheKey, &stats)
	switch err {
	case nil:
		contextLogger(c).Debug("serving stats from cache")
		c.JSON(http.StatusOK, stats)
		if tx := apm.TransactionFromContext(c.Request.Context()); tx != nil {
			tx.Context.SetTag("served_from_cache", "true")
		}
		return
	case persistence.ErrCacheMiss:
		// fetch and cache below
		if tx := apm.TransactionFromContext(c.Request.Context()); tx != nil {
			tx.Context.SetTag("served_from_cache", "false")
		}
		break
	default:
		err := errors.Wrap(err, "failed to get stats from cache")
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	stats, err = getStats(c.Request.Context(), h.db)
	if err != nil {
		err := errors.Wrap(err, "failed to query stats")
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if err := cache.Set(cacheKey, stats, time.Minute); err != nil {
		err := errors.Wrap(err, "failed to cache stats")
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	contextLogger(c).Debug("cached stats")
	c.JSON(http.StatusOK, stats)
}

func (h apiHandlers) getProducts(c *gin.Context) {
	products, err := getProducts(c.Request.Context(), h.db)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, products)
}

func (h apiHandlers) getTopProducts(c *gin.Context) {
	products, err := getTopProducts(c.Request.Context(), h.db)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, products)
}

func (h apiHandlers) getProductDetails(c *gin.Context) {
	idString := c.Param("id")
	if idString == "top" {
		products, err := getTopProducts(c.Request.Context(), h.db)
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
		err := errors.Wrap(err, "failed to parse product ID")
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	product, err := getProduct(c.Request.Context(), h.db, id)
	if err != nil {
		err := errors.Wrap(err, "failed to get product")
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
	customers, err := getProductCustomers(c.Request.Context(), h.db, id, limit)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, customers)
}

func (h apiHandlers) getProductTypes(c *gin.Context) {
	productTypes, err := getProductTypes(c.Request.Context(), h.db)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, productTypes)
}

func (h apiHandlers) getProductTypeDetails(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		err := errors.Wrap(err, "failed to parse product type ID")
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	productType, err := getProductType(c.Request.Context(), h.db, id)
	if err != nil {
		err := errors.Wrap(err, "failed to get product type details")
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if productType == nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.JSON(http.StatusOK, productType)
}

func (h apiHandlers) getCustomers(c *gin.Context) {
	customers, err := getCustomers(c.Request.Context(), h.db)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, customers)
}

func (h apiHandlers) getCustomerDetails(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		err := errors.Wrap(err, "failed to parse customer ID")
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	customer, err := getCustomer(c.Request.Context(), h.db, id)
	if err != nil {
		err := errors.Wrap(err, "failed to get customer details")
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
	orders, err := getOrders(c.Request.Context(), h.db)
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
	customer, err := getOrder(c.Request.Context(), h.db, id)
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

func (h apiHandlers) postOrder(c *gin.Context) {
	type line struct {
		ID     int `json:"id" binding:"required"`
		Amount int `json:"amount" binding:"required"`
	}
	var order struct {
		CustomerID int    `json:"customer_id" binding:"required"`
		Lines      []line `json:"lines" binding:"required"`
	}
	if err := c.BindJSON(&order); err != nil {
		return
	}
	lines := make([]ProductOrderLine, len(order.Lines))
	for i, line := range order.Lines {
		lines[i] = ProductOrderLine{
			Product: Product{ID: line.ID},
			Amount:  line.Amount,
		}
	}
	h.postOrderCommon(c, order.CustomerID, lines)
}

func (h apiHandlers) postOrderCSV(c *gin.Context) {
	customerID, err := strconv.Atoi(c.PostForm("customer"))
	if err != nil {
		err := errors.Wrap(err, "failed to parse customer ID")
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		err := errors.Wrap(err, "failed get CSV file")
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		err := errors.Wrap(err, "failed open CSV file")
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer file.Close()

	var lines []ProductOrderLine
	r := csv.NewReader(file)
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			err := errors.Wrap(err, "failed to parse CSV file")
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		productID, err := strconv.Atoi(record[0])
		if err != nil {
			err := errors.Wrap(err, "failed to parse product ID")
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		amount, err := strconv.Atoi(record[1])
		if err != nil {
			err := errors.Wrap(err, "failed to parse order amount")
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		lines = append(lines, ProductOrderLine{
			Product: Product{ID: productID},
			Amount:  amount,
		})
	}
	h.postOrderCommon(c, customerID, lines)
}

func (h apiHandlers) postOrderCommon(c *gin.Context, customerID int, lines []ProductOrderLine) {
	customer, err := getCustomer(c.Request.Context(), h.db, customerID)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	orderID, err := createOrder(c.Request.Context(), h.db, customer, lines)
	if err != nil {
		err := errors.Wrap(err, "failed to create order")
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if tx := apm.TransactionFromContext(c.Request.Context()); tx != nil {
		tx.Context.SetTag("customer_name", customer.FullName)
		tx.Context.SetTag("customer_email", customer.Email)
	}
	c.JSON(http.StatusOK, gin.H{"id": orderID})
}
