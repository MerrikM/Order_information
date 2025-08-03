package handler

import (
	"Order_information/internal/service"
	"Order_information/util"
	"log"
	"net/http"
)

type OrderHandler struct {
	orderService *service.OrderService
}

type OrderRequest struct {
	OrderUID string `json:"order_uid"`
}

func NewOrderHandler(orderService *service.OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService}
}

func (h *OrderHandler) GetOrderByUUID(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()

	orderUID := request.URL.Query().Get("order_uid")
	if orderUID == "" {
		util.RespondWithError(writer, http.StatusBadRequest, "параметр order_uid обязателен")
		return
	}

	order, err := h.orderService.GetOrderByUUID(ctx, orderUID)
	if err != nil {
		log.Printf("ошибка получения заказа: %v", err)
		util.RespondWithError(writer, http.StatusInternalServerError, "ошибка получения заказа")
		return
	}
	if order == nil {
		log.Printf("заказ не найден: %v", order)
		util.RespondWithError(writer, http.StatusNotFound, "заказ не найден")
		return
	}

	util.RespondWithJSON(writer, http.StatusOK, order)
}
