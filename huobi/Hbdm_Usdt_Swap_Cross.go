package huobi

import (
	"errors"
	"fmt"
	. "github.com/nntaoli-project/goex"
	"github.com/nntaoli-project/goex/internal/logger"
	"net/url"
	"time"
)

type HbdmUsdtSwapCross struct {
	base *Hbdm
	c    *APIConfig
}

const (
	usdtSwapCrossGetAccountInfoApiPath = "/linear-swap-api/v1/swap_cross_account_info"
	usdtSwapCrossPlacepOrderApiPath    = "/linear-swap-api/v1/swap_cross_order"
	usdtSwapCrossGetOrderInfoApiPath   = "/linear-swap-api/v1/swap_cross_order_info"
	usdtSwapCrossCancelOrderApiPath    = "/linear-swap-api/v1/swap_cross_cancel"
	usdtSwapCrossGetPositionApiPath    = "/linear-swap-api/v1/swap_cross_position_info"
	usdtSwapCrossCancelAllOrderApiPath = "/linear-swap-api/v1/swap_cross_cancelall"
)

func NewHbdmUsdtSwapCross(c *APIConfig) *HbdmUsdtSwapCross {
	if c.Lever <= 0 {
		c.Lever = 10
	}

	return &HbdmUsdtSwapCross{
		base: NewHbdm(c),
		c:    c,
	}
}

func (swap *HbdmUsdtSwapCross) GetExchangeName() string {
	return HBDM_USDT_SWAP_CROSS
}
func (swap *HbdmUsdtSwapCross) GetAccountInfo() (*FutureAccount, error) {

	type contractDetail struct {
		Symbol           string  `json:"symbol"`
		ContractCode     string  `json:"contract_code"`
		MarginPosition   float64  `json:"margin_position"`
		MarginFrozen     float64  `json:"margin_frozen"`
		MarginAvailable  float64 `json:"margin_available"`
		ProfitUnreal     float64 `json:"profit_unreal"`
		LiquidationPrice float64 `json:"liquidation_price"`
		LeverRate        float64 `json:"lever_rate"`
	}

	var accountInfoResponse []struct {
		MarginBalance   float64          `json:"margin_balance"`
		MarginPosition  float64          `json:"margin_position"`
		MarginFrozen    float64          `json:"margin_frozen"`
		MarginAvailable float64          `json:"margin_available"`
		ProfitReal      float64          `json:"profit_real"`
		ProfitUnreal    float64          `json:"profit_unreal"`
		RiskRate        float64          `json:"risk_rate"`
		ContractDetail  []contractDetail `json:"contract_detail"`
	}

	param := url.Values{}
	param.Set("margin_account", "USDT")

	err := swap.base.doRequest(usdtSwapCrossGetAccountInfoApiPath, &param, &accountInfoResponse)
	if err != nil {
		return nil, err
	}

	var futureAccount FutureAccount
	futureAccount.FutureSubAccounts = make(map[Currency]FutureSubAccount, 4)

	for _, acc := range accountInfoResponse {
		for _, detail := range acc.ContractDetail {
			currency := NewCurrency(detail.Symbol, "")
			futureAccount.FutureSubAccounts[currency] = FutureSubAccount{
				Currency:      currency,
				AccountRights: acc.MarginBalance,
				KeepDeposit:   acc.MarginPosition,
				ProfitReal:    acc.ProfitReal,
				ProfitUnreal:  detail.ProfitUnreal,
				RiskRate:      acc.RiskRate,
			}
		}

	}
	return &futureAccount, nil

}

func (swap *HbdmUsdtSwapCross) PlaceFutureOrder(currencyPair CurrencyPair, price float64, amount float64, openType int, orderPriceType OrderPriceType, leverRate int) (string, error) {

	priceStr := fmt.Sprint(price)

	if price == 0 {
		priceStr = ""
	}

	fmt.Println(price,priceStr)

	param := url.Values{}
	param.Set("contract_code", currencyPair.ToSymbol("-"))
	param.Set("client_order_id", fmt.Sprint(time.Now().UnixNano()))
	param.Set("price", priceStr)
	param.Set("volume", fmt.Sprint(amount))
	param.Set("lever_rate", fmt.Sprintf("%v", leverRate))

	direction, offset := swap.base.adaptOpenType(openType)
	param.Set("direction", direction)
	param.Set("offset", offset)
	logger.Info(direction, offset)

	param.Set("order_price_type", string(orderPriceType))

	var orderResponse struct {
		OrderId       string `json:"order_id_str"`
		ClientOrderId int64  `json:"client_order_id"`
	}

	err := swap.base.doRequest(usdtSwapCrossPlacepOrderApiPath, &param, &orderResponse)
	if err != nil {
		return "", err
	}

	return orderResponse.OrderId, nil
}

func (swap *HbdmUsdtSwapCross) FutureCancelOrder(currencyPair CurrencyPair, orderId string) (bool, error) {
	param := url.Values{}
	param.Set("order_id", orderId)
	param.Set("contract_code", currencyPair.ToSymbol("-"))

	var cancelResponse struct {
		Errors []struct {
			ErrMsg    string `json:"err_msg"`
			Successes string `json:"successes,omitempty"`
		} `json:"errors"`
	}

	err := swap.base.doRequest(usdtSwapCrossCancelOrderApiPath, &param, &cancelResponse)
	if err != nil {
		return false, err
	}

	if len(cancelResponse.Errors) > 0 {
		return false, errors.New(cancelResponse.Errors[0].ErrMsg)
	}

	return true, nil
}

func (swap *HbdmUsdtSwapCross) FutureCancelAll(currencyPair CurrencyPair, openType int) (bool, error) {
	param := url.Values{}
	param.Set("contract_code", currencyPair.ToSymbol("-"))
	if openType != 0 {
		direction, offset := swap.base.adaptOpenType(openType)
		param.Set("direction", direction)
		param.Set("offset", offset)
	}

	var cancelResponse struct {
		Errors  []string `json:"errors"`
		Success []string `json:"success"`
	}

	err := swap.base.doRequest(usdtSwapCrossCancelAllOrderApiPath, &param, &cancelResponse)
	if err != nil {
		return false, err
	}

	if len(cancelResponse.Errors) > 0 {
		return false, errors.New(cancelResponse.Errors[0])
	}
	return true, nil
}

//
func (swap *HbdmUsdtSwapCross) GetFuturePosition(currencyPair CurrencyPair) (*FuturePosition, error) {
	param := url.Values{}
	param.Set("contract_code", currencyPair.ToSymbol("-"))

	var (
		futuresPosition  *FuturePosition
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
			LastPrice      float64 `json:"last_price"`
		}
	)

	err := swap.base.doRequest(usdtSwapCrossGetPositionApiPath, &param, &positionResponse)
	if err != nil {
		return nil, err
	}

	futuresPosition = new(FuturePosition)

	for _, pos := range positionResponse {

		switch pos.Direction {
		case "sell":
			futuresPosition.ContractType = pos.ContractCode
			futuresPosition.Symbol = NewCurrencyPair3(pos.ContractCode, "-")
			futuresPosition.SellAmount = pos.Volume
			futuresPosition.SellAvailable = pos.Available
			futuresPosition.SellPriceAvg = pos.CostOpen
			futuresPosition.SellPriceCost = pos.CostHold
			futuresPosition.SellProfitReal = pos.ProfitRate
			futuresPosition.SellProfit = pos.Profit
			futuresPosition.LastPrice = pos.LastPrice
		case "buy":
			futuresPosition.ContractType = pos.ContractCode
			futuresPosition.Symbol = NewCurrencyPair3(pos.ContractCode, "-")
			futuresPosition.BuyAmount = pos.Volume
			futuresPosition.BuyAvailable = pos.Available
			futuresPosition.BuyPriceAvg = pos.CostOpen
			futuresPosition.BuyPriceCost = pos.CostHold
			futuresPosition.BuyProfitReal = pos.ProfitRate
			futuresPosition.BuyProfit = pos.Profit
			futuresPosition.LastPrice = pos.LastPrice
		}
	}

	return futuresPosition, nil
}

func (swap *HbdmUsdtSwapCross) GetFutureOrder(orderId string, currencyPair CurrencyPair) (*FutureOrder, error) {
	var (
		orderInfoResponse []OrderInfo
		param             = url.Values{}
	)

	param.Set("contract_code", currencyPair.ToSymbol("-"))
	param.Set("order_id", orderId)

	err := swap.base.doRequest(usdtSwapCrossGetOrderInfoApiPath, &param, &orderInfoResponse)
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

//
//
//func (swap *HbdmUsdtSwapCross) GetUnfinishFutureOrders(currencyPair CurrencyPair, contractType string) ([]FutureOrder, error) {
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
