package huobi

import (
	"errors"
	"fmt"
	. "github.com/nntaoli-project/goex"
	"github.com/nntaoli-project/goex/internal/logger"
	"net/url"
	"time"
)

type HbdmUsdtSwap struct {
	base *Hbdm
	c    *APIConfig
}

const (
	usdtSwapGetAccountInfoApiPath  = "/linear-swap-api/v1/swap_account_info"
	usdtSwapPlacepOrderApiPath     = "/linear-swap-api/v1/swap_order"
	usdtSwapgetOrderInfoApiPath    = "/linear-swap-api/v1/swap_order_info"
	usdtSwaplightningClosePosition = "/linear-swap-api/v1/swap_lightning_close_position"
	usdtSwapCancelOrderApiPath     = "/linear-swap-api/v1/swap_cancel"
	usdtSwapGetPositionApiPath     = "/linear-swap-api/v1/swap_position_info"
	usdtSwapCancelAllOrderApiPath     = "/linear-swap-api/v1/swap_cancelall"
)

func NewHbdmUsdtSwap(c *APIConfig) *HbdmUsdtSwap {
	if c.Lever <= 0 {
		c.Lever = 10
	}

	return &HbdmUsdtSwap{
		base: NewHbdm(c),
		c:    c,
	}
}

func (swap *HbdmUsdtSwap) GetExchangeName() string {
	return HBDM_USDT_SWAP
}
func (swap *HbdmUsdtSwap) GetAccountInfo(currencyPair ...CurrencyPair) (*FutureAccount, error) {
	var accountInfoResponse []struct {
		Symbol           string  `json:"symbol"`
		MarginBalance    float64 `json:"margin_balance"`
		MarginPosition   float64 `json:"margin_position"`
		MarginFrozen     float64 `json:"margin_frozen"`
		MarginAvailable  float64 `json:"margin_available"`
		ProfitReal       float64 `json:"profit_real"`
		ProfitUnreal     float64 `json:"profit_unreal"`
		RiskRate         float64 `json:"risk_rate"`
		LiquidationPrice float64 `json:"liquidation_price"`
	}

	param := url.Values{}
	if len(currencyPair) > 0 {
		param.Set("contract_code", currencyPair[0].ToSymbol("-"))
	}

	err := swap.base.doRequest(usdtSwapGetAccountInfoApiPath, &param, &accountInfoResponse)
	if err != nil {
		return nil, err
	}

	var futureAccount FutureAccount
	futureAccount.FutureSubAccounts = make(map[Currency]FutureSubAccount, 4)

	for _, acc := range accountInfoResponse {
		currency := NewCurrency(acc.Symbol, "")
		futureAccount.FutureSubAccounts[currency] = FutureSubAccount{
			Currency:      currency,
			AccountRights: acc.MarginBalance,
			KeepDeposit:   acc.MarginPosition,
			ProfitReal:    acc.ProfitReal,
			ProfitUnreal:  acc.ProfitUnreal,
			RiskRate:      acc.RiskRate,
		}
	}

	return &futureAccount, nil

}

func (swap *HbdmUsdtSwap) PlaceFutureOrder(currencyPair CurrencyPair, price float64, amount float64, openType int, orderPriceType string, leverRate int) (string, error) {
	param := url.Values{}
	param.Set("contract_code", currencyPair.ToSymbol("-"))
	param.Set("client_order_id", fmt.Sprint(time.Now().UnixNano()))
	param.Set("price", fmt.Sprint(price))
	param.Set("volume", fmt.Sprint(amount))
	param.Set("lever_rate", fmt.Sprintf("%v", leverRate))

	direction, offset := swap.base.adaptOpenType(openType)
	param.Set("direction", direction)
	param.Set("offset", offset)
	logger.Info(direction, offset)

	param.Set("order_price_type", orderPriceType)

	var orderResponse struct {
		OrderId       string `json:"order_id_str"`
		ClientOrderId int64  `json:"client_order_id"`
	}

	err := swap.base.doRequest(usdtSwapPlacepOrderApiPath, &param, &orderResponse)
	if err != nil {
		return "", err
	}

	return orderResponse.OrderId, nil
}

func (swap *HbdmUsdtSwap) FutureCancelOrder(currencyPair CurrencyPair, orderId string) (bool, error) {
	param := url.Values{}
	param.Set("order_id", orderId)
	param.Set("contract_code", currencyPair.ToSymbol("-"))

	var cancelResponse struct {
		Errors []struct {
			ErrMsg    string `json:"err_msg"`
			Successes string `json:"successes,omitempty"`
		} `json:"errors"`
	}

	err := swap.base.doRequest(usdtSwapCancelOrderApiPath, &param, &cancelResponse)
	if err != nil {
		return false, err
	}

	if len(cancelResponse.Errors) > 0 {
		return false, errors.New(cancelResponse.Errors[0].ErrMsg)
	}

	return true, nil
}

func (swap *HbdmUsdtSwap) FutureCancelAll(currencyPair CurrencyPair, openType int) (bool, error) {
	param := url.Values{}
	param.Set("contract_code", currencyPair.ToSymbol("-"))
	if openType != 0{
		direction, offset := swap.base.adaptOpenType(openType)
		param.Set("direction", direction)
		param.Set("offset", offset)
	}

	var cancelResponse struct {
		Errors []string `json:"errors"`
		Success []string `json:"success"`
	}

	err := swap.base.doRequest(usdtSwapCancelAllOrderApiPath, &param, &cancelResponse)
	if err != nil {
		return false, err
	}

	if len(cancelResponse.Errors) > 0 {
		return false, errors.New(cancelResponse.Errors[0])
	}
	return true, nil
}

//
func (swap *HbdmUsdtSwap) GetFuturePosition(currencyPair CurrencyPair) ([]FuturePosition, error) {
	param := url.Values{}
	param.Set("contract_code", currencyPair.ToSymbol("-"))

	var (
		tempPositionMap  map[string]*FuturePosition
		futuresPositions []FuturePosition
		positionResponse []struct {
			Symbol         string
			ContractCode   string  `json:"contract_code"`
			Volume         float64 `json:"volume"`
			Available      float64 `json:"available"`
			CostOpen       float64 `json:"cost_open"`
			CostHold       float64 `json:"cost_hold"`
			ProfitUnreal   float64 `json:"profit_unreal"`
			ProfitRate     float64 `json:"profit_rate"`
			Profit         float64 `json:"profit"`
			PositionMargin float64 `json:"position_margin"`
			LeverRate      float64 `json:"lever_rate"`
			Direction      string  `json:"direction"`
		}
	)

	err := swap.base.doRequest(usdtSwapGetPositionApiPath, &param, &positionResponse)
	if err != nil {
		return nil, err
	}

	futuresPositions = make([]FuturePosition, 0, 2)
	tempPositionMap = make(map[string]*FuturePosition, 2)

	for _, pos := range positionResponse {
		if tempPositionMap[pos.ContractCode] == nil {
			tempPositionMap[pos.ContractCode] = new(FuturePosition)
		}
		switch pos.Direction {
		case "sell":
			tempPositionMap[pos.ContractCode].ContractType = pos.ContractCode
			tempPositionMap[pos.ContractCode].Symbol = NewCurrencyPair3(pos.ContractCode, "-")
			tempPositionMap[pos.ContractCode].SellAmount = pos.Volume
			tempPositionMap[pos.ContractCode].SellAvailable = pos.Available
			tempPositionMap[pos.ContractCode].SellPriceAvg = pos.CostOpen
			tempPositionMap[pos.ContractCode].SellPriceCost = pos.CostHold
			tempPositionMap[pos.ContractCode].SellProfitReal = pos.ProfitRate
			tempPositionMap[pos.ContractCode].SellProfit = pos.Profit
		case "buy":
			tempPositionMap[pos.ContractCode].ContractType = pos.ContractCode
			tempPositionMap[pos.ContractCode].Symbol = NewCurrencyPair3(pos.ContractCode, "-")
			tempPositionMap[pos.ContractCode].BuyAmount = pos.Volume
			tempPositionMap[pos.ContractCode].BuyAvailable = pos.Available
			tempPositionMap[pos.ContractCode].BuyPriceAvg = pos.CostOpen
			tempPositionMap[pos.ContractCode].BuyPriceCost = pos.CostHold
			tempPositionMap[pos.ContractCode].BuyProfitReal = pos.ProfitRate
			tempPositionMap[pos.ContractCode].BuyProfit = pos.Profit
		}
	}

	for _, pos := range tempPositionMap {
		futuresPositions = append(futuresPositions, *pos)
	}

	return futuresPositions, nil
}

func (swap *HbdmUsdtSwap) GetFutureOrder(orderId string, currencyPair CurrencyPair) (*FutureOrder, error) {
	var (
		orderInfoResponse []OrderInfo
		param             = url.Values{}
	)

	param.Set("contract_code", currencyPair.ToSymbol("-"))
	param.Set("order_id", orderId)

	err := swap.base.doRequest(usdtSwapgetOrderInfoApiPath, &param, &orderInfoResponse)
	if err != nil {
		return nil, err
	}

	if len(orderInfoResponse) == 0 {
		return nil, errors.New("not found")
	}

	orderInfo := orderInfoResponse[0]

	return &FutureOrder{
		Currency:     currencyPair,
		ClientOid:    fmt.Sprint(orderInfo.ClientOrderId),
		OrderID2:     fmt.Sprint(orderInfo.OrderId),
		Price:        orderInfo.Price,
		Amount:       orderInfo.Volume,
		AvgPrice:     orderInfo.TradeAvgPrice,
		DealAmount:   orderInfo.TradeVolume,
		OrderID:      orderInfo.OrderId,
		Status:       swap.base.adaptOrderStatus(orderInfo.Status),
		OType:        swap.base.adaptOffsetDirectionToOpenType(orderInfo.Offset, orderInfo.Direction),
		LeverRate:    orderInfo.LeverRate,
		Fee:          orderInfo.Fee,
		ContractName: orderInfo.ContractCode,
		OrderTime:    orderInfo.CreatedAt,
	}, nil
}

func (swap *HbdmUsdtSwap) LightningClose(currencyPair CurrencyPair, volume float64, openType int) (string, error) {
	param := url.Values{}
	param.Set("contract_code", currencyPair.ToSymbol("-"))
	param.Set("client_order_id", fmt.Sprint(time.Now().UnixNano()))
	param.Set("volume", fmt.Sprint(volume))
	direction, _ := swap.base.adaptOpenType(openType)
	param.Set("direction", fmt.Sprint(direction))

	var orderResponse struct {
		OrderId       string `json:"order_id_str"`
		ClientOrderId int64  `json:"client_order_id"`
	}

	err := swap.base.doRequest(usdtSwaplightningClosePosition, &param, &orderResponse)
	if err != nil {
		return "", err
	}

	return orderResponse.OrderId, nil
}

//
//
//func (swap *HbdmUsdtSwap) GetUnfinishFutureOrders(currencyPair CurrencyPair, contractType string) ([]FutureOrder, error) {
//	param := url.Values{}
//	param.Set("contract_code", currencyPair.ToSymbol("-"))
//	param.Set("page_size", "50")
//
//	var openOrderResponse struct {
//		Orders []OrderInfo
//	}
//
//	err := swap.base.doRequest(getOpenOrdersApiPath, &param, &openOrderResponse)
//	if err != nil {
//		return nil, err
//	}
//
//	openOrders := make([]FutureOrder, 0, len(openOrderResponse.Orders))
//	for _, ord := range openOrderResponse.Orders {
//		openOrders = append(openOrders, FutureOrder{
//			Currency:   currencyPair,
//			ClientOid:  fmt.Sprint(ord.ClientOrderId),
//			OrderID2:   fmt.Sprint(ord.OrderId),
//			Price:      ord.Price,
//			Amount:     ord.Volume,
//			AvgPrice:   ord.TradeAvgPrice,
//			DealAmount: ord.TradeVolume,
//			OrderID:    ord.OrderId,
//			Status:     swap.base.adaptOrderStatus(ord.Status),
//			OType:      swap.base.adaptOffsetDirectionToOpenType(ord.Offset, ord.Direction),
//			LeverRate:  ord.LeverRate,
//			Fee:        ord.Fee,
//			OrderTime:  ord.CreatedAt,
//		})
//	}
//
//	return openOrders, nil
//}
