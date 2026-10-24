package service

import (
	"testing"
)

func TestClassifyCategory(t *testing.T) {
	testCases := []struct {
		description string
		isIncome    bool
		expected    string
	}{
		{"Комиссия за ведение счета за сентябрь 2024 г.", false, "Банковские комиссии"},
		{"Оплата по УПД № 279 от 19.08.2024. НДС не облагается", false, "Поставщики"},
		{"Зачисление по операциям СБП за 01.10.2024", true, "Выручка"},
		{"Поступление от покупателя по счету № 12", true, "Выручка"},
		{"Выплата заработной платы за 1 половину месяца", false, "Зарплата"},
		{"Аванс сотрудникам по реестру № 5", false, "Зарплата"},
		{"Оплата аренды складского помещения за октябрь", false, "Аренда"},
		{"Оплата за аренду офиса", false, "Аренда"},
		{"Единый налоговый платеж ЕНП", false, "Налоги и взносы"},
		{"Оплата страховых взносов за работников", false, "Налоги и взносы"},
		{"Оплата услуг перевозки груза автотранспортом", false, "Логистика"},
		{"Оплата услуг связи и интернет за месяц", false, "Связь и IT"},
		{"Оплата подписки на облачный сервис", false, "Связь и IT"},
		{"Погашение процентов по кредитному договору", false, "Кредиты и займы"},
		{"Оплата за электроэнергию по договору энергоснабжения", false, "Коммунальные"},
		{"Пополнение рекламного кабинета Яндекс Директ", false, "Реклама и маркетинг"},
		{"Приобретение канцтоваров и бумаги для офиса", false, "Офисные и хозтовары"},
		{"Оплата бухгалтерских услуг по договору", false, "Административные расходы"},
		{"Перевод собственных средств на карту", false, "Переводы между своими счетами"},
		{"Возврат денежных средств покупателю", false, "Возвраты"},
		{"Непонятная операция без ключевых слов", false, "Прочее"},
	}

	for _, tc := range testCases {
		cat := ClassifyCategory(tc.description, tc.isIncome)
		if cat != tc.expected {
			t.Errorf("Desc '%s' (isIncome=%v): expected '%s', got '%s'",
				tc.description, tc.isIncome, tc.expected, cat)
		} else {
			t.Logf("PASS: '%s' -> '%s'", tc.description, cat)
		}
	}
}

func TestClassifyCategoryDirectionRestriction(t *testing.T) {
	// "Зарплата" is only allowed for expense (isIncome=false). If isIncome=true, it shouldn't be classified as Зарплата!
	cat := ClassifyCategory("Зарплата от заказчика", true)
	if cat == "Зарплата" {
		t.Errorf("expected Зарплата to be restricted to expenses, got %s", cat)
	}

	// "Выручка" is only allowed for income (isIncome=true).
	catOut := ClassifyCategory("Выручка от покупателя", false)
	if catOut == "Выручка" {
		t.Errorf("expected Выручка to be restricted to income, got %s", catOut)
	}
}
