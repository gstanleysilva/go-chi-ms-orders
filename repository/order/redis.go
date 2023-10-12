package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/guhstanley/go-chi-ms-orders/model"
	"github.com/redis/go-redis/v9"
)

type RedisRepo struct {
	Client *redis.Client
}

func orderIDKey(id uint64) string {
	return fmt.Sprintf("order:%d", id)
}

func (r *RedisRepo) Insert(ctx context.Context, order model.Order) error {
	//Transforms the order into a json string
	data, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to encode order: %w", err)
	}

	//Create a transaction
	txn := r.Client.TxPipeline()

	key := orderIDKey(order.OrderID)

	//SetNX sets the data only if it still does not exist
	res := txn.SetNX(ctx, key, string(data), 0)
	if err := res.Err(); err != nil {
		txn.Discard()
		return fmt.Errorf("failed to set: %w", err)
	}

	//Add the key of the order into a Set that will be used for pagination
	if err := txn.SAdd(ctx, "orders", key).Err(); err != nil {
		txn.Discard()
		return fmt.Errorf("failed to add to orders set: %w", err)
	}

	//Executes the transaction
	if _, err := txn.Exec(ctx); err != nil {
		return fmt.Errorf("failed to exec: %w", err)
	}

	return nil
}

var ErrNotExist = errors.New("order does not exist")

func (r *RedisRepo) FindByID(ctx context.Context, id uint64) (*model.Order, error) {
	key := orderIDKey(id)

	//Get the json value from the key
	value, err := r.Client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, ErrNotExist
	} else if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}

	//Decode the json into the object to be returned
	var order model.Order
	err = json.Unmarshal([]byte(value), &order)
	if err != nil {
		return nil, fmt.Errorf("failed to decode order json: %w", err)
	}

	return &order, nil
}

func (r *RedisRepo) DeleteByID(ctx context.Context, id uint64) error {
	key := orderIDKey(id)

	//Create a transaction
	txn := r.Client.TxPipeline()

	//Executes the deletion and checks the errors
	err := txn.Del(ctx, key).Err()
	if errors.Is(err, redis.Nil) {
		txn.Discard()
		return ErrNotExist
	} else if err != nil {
		txn.Discard()
		return fmt.Errorf("get order: %w", err)
	}

	//Remove the order key from the Set
	if err := txn.SRem(ctx, "orders", key).Err(); err != nil {
		txn.Discard()
		return fmt.Errorf("failed to remove from orders set: %w", err)
	}

	//Executes the transaction
	if _, err := txn.Exec(ctx); err != nil {
		return fmt.Errorf("failed to exec: %w", err)
	}

	return nil
}

func (r *RedisRepo) Update(ctx context.Context, order model.Order) error {
	data, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to encode order: %w", err)
	}

	key := orderIDKey(order.OrderID)

	//Set value only if it already exist on redis
	err = r.Client.SetXX(ctx, key, string(data), 0).Err()
	if errors.Is(err, redis.Nil) {
		return ErrNotExist
	} else if err != nil {
		return fmt.Errorf("get order: %w", err)
	}

	return nil
}

// Pagination structure
type FindAllPage struct {
	Size   uint
	Offset uint
}

// Structure containing the values and the cursor from where we stopped
type FindResult struct {
	Orders []model.Order
	Cursor uint64
}

func (r *RedisRepo) FindAll(ctx context.Context, page FindAllPage) (*FindResult, error) {
	//Get all order keys from the set using the pagination
	res := r.Client.SScan(ctx, "orders", uint64(page.Offset), "*", int64(page.Size))

	//Execute the scan defined above
	keys, cursor, err := res.Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get orders ids: %w", err)
	}

	//Check if no values was found
	if len(keys) == 0 {
		return &FindResult{
			Orders: []model.Order{},
		}, nil
	}

	//Execute a multi get by all keys returned from the set
	xs, err := r.Client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}

	//Create an slice of orders to hold the return
	orders := make([]model.Order, len(xs))

	//Format the orders and return
	for i, x := range xs {
		x := x.(string)
		var order model.Order

		err := json.Unmarshal([]byte(x), &order)
		if err != nil {
			return nil, fmt.Errorf("failed to decode order json: %w", err)
		}

		orders[i] = order
	}

	return &FindResult{
		Orders: orders,
		Cursor: cursor,
	}, nil
}
