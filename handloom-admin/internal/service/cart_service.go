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

// NewCartService creates a new CartService
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

// cartPK derives the DynamoDB partition key for a customer's cart
func cartPK(customerID string) string {
	return "CART#" + customerID
}

// cartTTL returns a Unix timestamp 30 days from now
func cartTTL() int64 {
	return time.Now().Add(cartTTLDays * 24 * time.Hour).Unix()
}

// GetCart retrieves the cart for a customer
func (s *CartService) GetCart(ctx context.Context, customerID string) (*domain.CartWithItems, error) {
	result, err := s.cartRepo.GetCart(ctx, cartPK(customerID))
	if err != nil {
		return nil, err
	}
	return result, nil
}

// AddItem adds an item to the customer's cart after validating the product
func (s *CartService) AddItem(ctx context.Context, customerID string, req domain.AddCartItemRequest) (*domain.CartWithItems, error) {
	// Validate product exists and is ACTIVE
	product, err := s.productRepo.GetByID(ctx, req.ProductID)
	if err != nil {
		return nil, err
	}
	if product.Status != domain.ProductStatusActive {
		return nil, errors.BadRequest("Product is not available")
	}

	// Check stock availability
	inventory, err := s.inventoryRepo.GetByProductID(ctx, req.ProductID)
	if err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
		// No inventory record means no stock
		return nil, errors.New(errors.ErrCodeInsufficientStock, "Product is out of stock")
	}
	if inventory.AvailableQty < req.Quantity {
		return nil, errors.New(errors.ErrCodeInsufficientStock, "Insufficient stock available")
	}

	// Determine primary image
	productImage := primaryImage(product.Images)

	pk := cartPK(customerID)
	ttl := cartTTL()

	// Build the cart item
	item := &domain.CartItem{
		ProductID:    req.ProductID,
		ProductName:  product.Name,
		ProductSKU:   product.SKU,
		ProductImage: productImage,
		Quantity:     req.Quantity,
		UnitPrice:    product.SellingPrice,
		TotalPrice:   product.SellingPrice * int64(req.Quantity),
		IsCustomSize: req.Dimensions != nil,
		Dimensions:   req.Dimensions,
		QuoteID:      req.QuoteID,
		AddedAt:      time.Now(),
		TTL:          ttl,
	}
	item.SetKeys(pk)

	// Write the item
	if err := s.cartRepo.PutCartItem(ctx, item); err != nil {
		return nil, err
	}

	// Recalculate and update header
	if err := s.recalculateHeader(ctx, pk, customerID); err != nil {
		return nil, err
	}

	s.logger.WithContext(ctx).Infof("Added item %s to cart for customer %s", req.ProductID, customerID)

	return s.cartRepo.GetCart(ctx, pk)
}

// UpdateItemQuantity updates the quantity of a cart item
func (s *CartService) UpdateItemQuantity(ctx context.Context, customerID, productID string, quantity int) (*domain.CartWithItems, error) {
	// If quantity is 0, remove the item
	if quantity == 0 {
		return s.RemoveItem(ctx, customerID, productID)
	}

	// Validate stock availability
	inventory, err := s.inventoryRepo.GetByProductID(ctx, productID)
	if err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
		return nil, errors.New(errors.ErrCodeInsufficientStock, "Product is out of stock")
	}
	if inventory.AvailableQty < quantity {
		return nil, errors.New(errors.ErrCodeInsufficientStock, "Insufficient stock available")
	}

	pk := cartPK(customerID)

	// Get the current cart to find the item's unit price
	cart, err := s.cartRepo.GetCart(ctx, pk)
	if err != nil {
		return nil, err
	}

	var unitPrice int64
	found := false
	for _, item := range cart.Items {
		if item.ProductID == productID {
			unitPrice = item.UnitPrice
			found = true
			break
		}
	}
	if !found {
		return nil, errors.NotFound("Cart item not found")
	}

	totalPrice := unitPrice * int64(quantity)

	if err := s.cartRepo.UpdateCartItem(ctx, pk, productID, quantity, totalPrice); err != nil {
		return nil, err
	}

	// Recalculate and update header
	if err := s.recalculateHeader(ctx, pk, customerID); err != nil {
		return nil, err
	}

	s.logger.WithContext(ctx).Infof("Updated item %s quantity to %d for customer %s", productID, quantity, customerID)

	return s.cartRepo.GetCart(ctx, pk)
}

// RemoveItem removes an item from the cart
func (s *CartService) RemoveItem(ctx context.Context, customerID, productID string) (*domain.CartWithItems, error) {
	pk := cartPK(customerID)

	if err := s.cartRepo.DeleteCartItem(ctx, pk, productID); err != nil {
		return nil, err
	}

	// Recalculate and update header
	if err := s.recalculateHeader(ctx, pk, customerID); err != nil {
		return nil, err
	}

	s.logger.WithContext(ctx).Infof("Removed item %s from cart for customer %s", productID, customerID)

	return s.cartRepo.GetCart(ctx, pk)
}

// ClearCart removes all items from the cart
func (s *CartService) ClearCart(ctx context.Context, customerID string) error {
	pk := cartPK(customerID)

	if err := s.cartRepo.ClearCart(ctx, pk); err != nil {
		return err
	}

	s.logger.WithContext(ctx).Infof("Cleared cart for customer %s", customerID)

	return nil
}

// MergeGuestCart merges guest cart items into the customer's cart.
// For each item, if it already exists in the cart, keep the higher quantity.
func (s *CartService) MergeGuestCart(ctx context.Context, customerID string, items []domain.AddCartItemRequest) (*domain.CartWithItems, error) {
	pk := cartPK(customerID)

	// Get existing cart to check for duplicates
	existingCart, err := s.cartRepo.GetCart(ctx, pk)
	if err != nil {
		return nil, err
	}

	// Build a map of existing items by product ID
	existingItems := make(map[string]int)
	for _, item := range existingCart.Items {
		existingItems[item.ProductID] = item.Quantity
	}

	for _, req := range items {
		existingQty, exists := existingItems[req.ProductID]
		if exists && existingQty >= req.Quantity {
			// Existing cart already has equal or higher quantity, skip
			continue
		}

		// Add or replace with the guest item (which has higher quantity)
		_, err := s.AddItem(ctx, customerID, req)
		if err != nil {
			// Log but continue merging other items
			s.logger.WithContext(ctx).Warnf("Failed to merge item %s: %v", req.ProductID, err)
			continue
		}
	}

	s.logger.WithContext(ctx).Infof("Merged guest cart (%d items) for customer %s", len(items), customerID)

	return s.cartRepo.GetCart(ctx, pk)
}

// recalculateHeader reads all cart items and updates the cart header with
// the correct item count and subtotal.
func (s *CartService) recalculateHeader(ctx context.Context, pk, customerID string) error {
	cart, err := s.cartRepo.GetCart(ctx, pk)
	if err != nil {
		return err
	}

	var subtotal int64
	itemCount := len(cart.Items)
	for _, item := range cart.Items {
		subtotal += item.TotalPrice
	}

	header := cart.Cart
	header.CustomerID = customerID
	header.ItemCount = itemCount
	header.Subtotal = subtotal
	header.Currency = "INR"
	header.UpdatedAt = time.Now()
	header.TTL = cartTTL()
	header.EntityType = "CART"
	header.PK = pk
	header.SK = "METADATA"

	return s.cartRepo.UpdateCartHeader(ctx, header)
}

// primaryImage returns the URL of the primary image, or the first image URL,
// or an empty string if no images exist.
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

// Ensure interface compliance
var _ domain.CartService = (*CartService)(nil)
