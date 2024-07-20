package service

import (
	"errors"
	"shopping-cart/builder"
	"shopping-cart/model/database"
	"shopping-cart/model/datatransfer/order"
	"shopping-cart/repository"
	"time"
)

type OrderService interface {
	CreateOrder(orderRequest *order.Request) (*database.Order, error)
	GetOrderByID(id int) (*database.Order, error)
	UpdateOrderStatusAndNote(id int, orderRequest *order.StatusRequest) (*database.Order, error)
	DeleteOrder(id int) error
	ListAllOrders() ([]database.Order, error)
	ListHistoryOrdersByDisplayNameAndProductID(displayName string, productID int) ([]database.Order, error)
	GetOrderByUserDisplayName(displayName string) ([]database.Order, error)
}

type orderService struct {
	orderRepo   repository.OrderRepository
	productRepo repository.ProductRepository
	userRepo    repository.UserRepository
}

func NewOrderService(orderRepo repository.OrderRepository, productRepo repository.ProductRepository, userRepo repository.UserRepository) OrderService {
	return &orderService{
		orderRepo:   orderRepo,
		productRepo: productRepo,
		userRepo:    userRepo,
	}
}

func validateOrderRequest(s *orderService, orderRequest *order.Request) (float64, map[int]*database.Product, error) {
	totalPrice := 0.0
	productMap := make(map[int]*database.Product)

	productIDs := make([]int, 0, len(orderRequest.OrderDetails))
	for _, detail := range orderRequest.OrderDetails {
		if detail.Quantity <= 0 {
			return 0, nil, errors.New("Quantity must be greater than zero")
		}
		productIDs = append(productIDs, detail.ProductID)
	}

	products, err := s.productRepo.FindByIDs(productIDs)
	if err != nil {
		return 0, nil, err
	}

	for _, product := range products {
		productMap[product.ID] = product
	}

	for i, detail := range orderRequest.OrderDetails {
		product, exists := productMap[detail.ProductID]
		if !exists {
			return 0, nil, errors.New("Product not found or already sold out")
		}

		if product.Stock < detail.Quantity {
			return 0, nil, errors.New("Insufficient stock for product " + product.Name)
		}

		if time.Now().After(product.ExpirationTime) {
			return 0, nil, errors.New("Product " + product.Name + " is expired")
		}

		orderRequest.OrderDetails[i].Price = product.Price
		totalPrice += float64(detail.Quantity) * product.Price
	}

	return totalPrice, productMap, nil
}

func (s *orderService) CreateOrder(orderRequest *order.Request) (*database.Order, error) {
	totalPrice, productMap, err := validateOrderRequest(s, orderRequest)
	if err != nil {
		return nil, err
	}

	order := builder.NewOrderBuilder().
		SetUserID(orderRequest.UserID).
		SetTotalPrice(totalPrice).
		SetNote(orderRequest.Note).
		SetStatus("pending").
		SetOrderDetails(orderRequest.OrderDetails).
		Build()

	tx := s.orderRepo.BeginTransaction()

	err = s.orderRepo.Create(order)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	var productsToUpdate []*database.Product
	for _, detail := range order.OrderDetails {
		product := productMap[detail.ProductID]
		product.Stock -= detail.Quantity
		productsToUpdate = append(productsToUpdate, product)
	}

	err = s.productRepo.BatchUpdate(productsToUpdate)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()

	return order, nil
}

func (s *orderService) GetOrderByID(id int) (*database.Order, error) {
	return s.orderRepo.FindByID(id)
}

func (s *orderService) UpdateOrderStatusAndNote(id int, orderRequest *order.StatusRequest) (*database.Order, error) {
	order, err := s.orderRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("Order not found")
	}

	if orderRequest.Status != "" {
		order.Status = orderRequest.Status
	}

	if orderRequest.Note != "" {
		order.Note = orderRequest.Note
	}

	err = s.orderRepo.Update(order)
	if err != nil {
		return nil, err
	}

	return order, nil
}

func (s *orderService) DeleteOrder(id int) error {
	order, err := s.orderRepo.FindByID(id)
	if err != nil {
		return errors.New("Order not found")
	}

	tx := s.orderRepo.BeginTransaction()

	for _, detail := range order.OrderDetails {
		product, err := s.productRepo.InternalFindByID(detail.ProductID)
		if err != nil {
			tx.Rollback()
			return err
		}
		product.Stock += detail.Quantity
		err = s.productRepo.Update(product)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	err = s.orderRepo.SoftDelete(order)
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

func (s *orderService) ListAllOrders() ([]database.Order, error) {
	return s.orderRepo.FindAll()
}

func (s *orderService) ListHistoryOrdersByDisplayNameAndProductID(displayName string, productID int) ([]database.Order, error) {
	user, err := s.userRepo.FindByDisplayName(displayName)
	if err != nil {
		return nil, err
	}

	return s.orderRepo.FindByUserIDAndProductID(user.ID, productID)
}

func (s *orderService) GetOrderByUserDisplayName(displayName string) ([]database.Order, error) {
	user, err := s.userRepo.FindByDisplayName(displayName)
	if err != nil {
		return nil, err
	}

	return s.orderRepo.FindByUserID(user.ID)
}
