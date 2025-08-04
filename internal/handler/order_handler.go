package handler

import (
	"Order_information/internal/service"
	"Order_information/util"
	"errors"
	"github.com/go-chi/chi/v5"
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

	orderUID := chi.URLParam(request, "order_uid")
	if orderUID == "" {
		util.RespondWithError(writer, http.StatusBadRequest, "параметр order_uid обязателен")
		return
	}

	order, err := h.orderService.GetOrderByUUID(ctx, orderUID)
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			log.Println(err)
			util.RespondWithError(writer, http.StatusNotFound, "не удалось найти заказ")
			return
		}
		log.Println(err)
		util.RespondWithError(writer, http.StatusInternalServerError, "ошибка получения заказа")
		return
	}

	util.RespondWithJSON(writer, http.StatusOK, order)
}
