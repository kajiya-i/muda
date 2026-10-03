package ledger

import "github.com/kajiya-i/muda/backend/internal/domain/money"

// Balances は、移動の列から、各勘定の残高を求める。
// 勘定の残高は、その感情が移動先になった金額の合計あから、移動元になった金額の
// 合計を引いたもの（docs/domain/wallet.md）。残高は保存せず、毎回この関数で導出する。
func Balances(transfers []Transfer) (map[Account]money.Money, error) {
	balances := make(map[Account]money.Money)
	for _, t := range transfers {
		amount := t.amount.Money()

		to, err := balanceOrZero(balances, t.to).Add(amount)
		if err != nil {
			return nil, err
		}
		balances[t.to] = to

		from, err := balanceOrZero(balances, t.from).Sub(amount)
		if err != nil {
			return nil, err
		}
		balances[t.from] = from
	}
	return balances, nil
}

// BalanceOf は、移動の列から、指定した勘定の残高を求める。
func BalanceOf(transfers []Transfer, account Account) (money.Money, error) {
	balances, err := Balances(transfers)
	if err != nil {
		return money.Money{}, err
	}
	return balanceOrZero(balances, account), nil
}

func balanceOrZero(balances map[Account]money.Money, account Account) money.Money {
	if b, ok := balances[account]; ok {
		return b
	}
	return money.Zero(account.currency)
}
