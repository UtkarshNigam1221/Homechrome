package service

import (
	"context"
	"time"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
)

const cartTTLDays = 30

// CartService implements domain.CartService
type CartService struct {
	cartRepo      domain.CartRepository
	productRepo   domain.ProductRepository
	inventoryRepo domain.InventoryRepository
	logger        *logger.Logger
}

func NewCartService(
	cartRepo domain.CartRepository,
	productRepo domain.ProductRepository,
	inventoryRepo domain.InventoryRepository,
	logger *logger.Logger,
) *CartService {
	return &CartService{
		cartRepo:      cartRepo,
		productRepo:   productRepo,
		inventoryRepo: inventoryRepo,
		logger:        logger,
	}
}

func cartPK(customerID string) string {
	return "CART#" + customerID
}

func cartTTL() int64 {
	return time.Now().Add(cartTTLDays * 24 * time.Hour).Unix()
}

func (s *CartService) GetCart(ctx context.Context, cartOwner string, isGuest bool) (*domain.CartWithItems, error) {
	return s.cartRepo.GetCart(ctx, cartPK(cartOwner))
}

func (s *CartService) AddItem(ctx context.Context, cartOwner string, isGuest bool, req domain.AddCartItemRequest) (*domain.CartWithItems, error) {
	product, err := s.productRepo.GetByID(ctx, req.ProductID)
	if err != nil {
		return nil, err
	}
	if product.Status != domain.ProductStatusActive {
		return nil, errors.BadRequest("Product is not available")
	}

	if err := s.validateStock(ctx, req.ProductID, req.Quantity); err != nil {
		return nil, err
	}

	pk := cartPK(cartOwner)

	item := &domain.CartItem{
		ProductID:    req.ProductID,
		ProductName:  product.Name,
		ProductSKU:   product.SKU,
		ProductImage: primaryImage(product.Images),
		CategoryID:   product.CategoryID,
		Quantity:     req.Quantity,
		UnitPrice:    product.SellingPrice,
		TotalPrice:   product.SellingPrice * int64(req.Quantity),
		IsCustomSize: req.Dimensions != nil,
		Dimensions:   req.Dimensions,
		QuoteID:      req.QuoteID,
		AddedAt:      time.Now(),
		TTL:          cartTTL(),
	}
	item.SetKeys(pk)

	if err := s.cartRepo.PutCartItem(ctx, item); err != nil {
		return nil, err
	}

	s.logger.WithContext(ctx).Infof("Added item %s to cart for %s", req.ProductID, cartOwner)

	return s.recalculateAndGetCart(ctx, pk, cartOwner, isGuest)
}

func (s *CartService) UpdateItemQuantity(ctx context.Context, cartOwner string, isGuest bool, productID string, quantity int) (*domain.CartWithItems, error) {
	if quantity == 0 {
		return s.RemoveItem(ctx, cartOwner, isGuest, productID)
	}

	if err := s.validateStock(ctx, productID, quantity); err != nil {
		return nil, err
	}

	pk := cartPK(cartOwner)

	cart, err := s.cartRepo.GetCart(ctx, pk)
	if err != nil {
		return nil, err
	}

	unitPrice, found := findItemUnitPrice(cart.Items, productID)
	if !found {
		return nil, errors.NotFound("Cart item not found")
	}

	if err := s.cartRepo.UpdateCartItem(ctx, pk, productID, quantity, unitPrice*int64(quantity)); err != nil {
		return nil, err
	}

	s.logger.WithContext(ctx).Infof("Updated item %s quantity to %d for %s", productID, quantity, cartOwner)

	return s.recalculateAndGetCart(ctx, pk, cartOwner, isGuest)
}

func (s *CartService) RemoveItem(ctx context.Context, cartOwner string, isGuest bool, productID string) (*domain.CartWithItems, error) {
	pk := cartPK(cartOwner)

	if err := s.cartRepo.DeleteCartItem(ctx, pk, productID); err != nil {
		return nil, err
	}

	s.logger.WithContext(ctx).Infof("Removed item %s from cart for %s", productID, cartOwner)

	return s.recalculateAndGetCart(ctx, pk, cartOwner, isGuest)
}

func (s *CartService) ClearCart(ctx context.Context, cartOwner string) error {
	if err := s.cartRepo.ClearCart(ctx, cartPK(cartOwner)); err != nil {
		return err
	}

	s.logger.WithContext(ctx).Infof("Cleared cart for customer %s", cartOwner)
	return nil
}

// MergeGuestCart merges guest cart items into the customer's cart.
// For each item, if it already exists in the cart, keep the higher quantity.
func (s *CartService) MergeGuestCart(ctx context.Context, customerID string, items []domain.AddCartItemRequest) (*domain.CartWithItems, error) {
	pk := cartPK(customerID)

	existingCart, err := s.cartRepo.GetCart(ctx, pk)
	if err != nil {
		return nil, err
	}

	existingQty := make(map[string]int, len(existingCart.Items))
	for _, item := range existingCart.Items {
		existingQty[item.ProductID] = item.Quantity
	}

	for _, req := range items {
		if qty, exists := existingQty[req.ProductID]; exists && qty >= req.Quantity {
			continue
		}
		if _, err := s.AddItem(ctx, customerID, false, req); err != nil {
			s.logger.WithContext(ctx).Warnf("Failed to merge item %s: %v", req.ProductID, err)
			continue
		}
	}

	s.logger.WithContext(ctx).Infof("Merged guest cart (%d items) for customer %s", len(items), customerID)

	return s.cartRepo.GetCart(ctx, pk)
}

// MergeGuestSession reads items from a guest cart and merges them into the customer's cart.
func (s *CartService) MergeGuestSession(ctx context.Context, customerID, guestSessionID string) error {
	guestCart, err := s.cartRepo.GetCart(ctx, cartPK(guestSessionID))
	if err != nil {
		return err
	}

	if len(guestCart.Items) == 0 {
		return nil
	}

	items := make([]domain.AddCartItemRequest, len(guestCart.Items))
	for i, item := range guestCart.Items {
		items[i] = domain.AddCartItemRequest{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}

	if _, err := s.MergeGuestCart(ctx, customerID, items); err != nil {
		return err
	}

	return s.ClearCart(ctx, guestSessionID)
}

// validateStock checks that the product has sufficient available inventory.
func (s *CartService) validateStock(ctx context.Context, productID string, quantity int) error {
	inventory, err := s.inventoryRepo.GetByProductID(ctx, productID)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		return errors.New(errors.ErrCodeInsufficientStock, "Product is out of stock")
	}
	if inventory.AvailableQty < quantity {
		return errors.New(errors.ErrCodeInsufficientStock, "Insufficient stock available")
	}
	return nil
}

// recalculateAndGetCart reads all cart items, updates the header totals,
// and returns the final cart state — avoiding a redundant second read.
func (s *CartService) recalculateAndGetCart(ctx context.Context, pk, cartOwner string, isGuest bool) (*domain.CartWithItems, error) {
	cart, err := s.cartRepo.GetCart(ctx, pk)
	if err != nil {
		return nil, err
	}

	var subtotal int64
	for _, item := range cart.Items {
		subtotal += item.TotalPrice
	}

	header := cart.Cart
	if isGuest {
		header.SessionID = cartOwner
		header.CustomerID = ""
	} else {
		header.CustomerID = cartOwner
	}
	header.ItemCount = len(cart.Items)
	header.Subtotal = subtotal
	header.Currency = "INR"
	header.UpdatedAt = time.Now()
	header.TTL = cartTTL()
	header.EntityType = "CART"
	header.PK = pk
	header.SK = "METADATA"

	if err := s.cartRepo.UpdateCartHeader(ctx, header); err != nil {
		return nil, err
	}

	cart.Cart = header
	return cart, nil
}

func findItemUnitPrice(items []domain.CartItem, productID string) (int64, bool) {
	for _, item := range items {
		if item.ProductID == productID {
			return item.UnitPrice, true
		}
	}
	return 0, false
}

func primaryImage(images []domain.ProductImage) string {
	if len(images) == 0 {
		return ""
	}
	for _, img := range images {
		if img.IsPrimary {
			return img.URL
		}
	}
	return images[0].URL
}

var _ domain.CartService = (*CartService)(nil)
