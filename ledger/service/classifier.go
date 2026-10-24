package service

import (
	"regexp"
	"strings"
)

type categoryRule struct {
	category        string
	regex           *regexp.Regexp
	allowIncome     bool
	allowExpense    bool
}

var categoryRules []categoryRule

func init() {
	// Rules ordered by priority from Category.ipynb
	rawRules := []struct {
		category     string
		pattern      string
		allowIncome  bool
		allowExpense bool
	}{
		{
			category:     "Переводы между своими счетами",
			pattern:      `(?i)(перевод.*собственн|собственн.*средств|между\s*своими|между\s*счетами|перевод.*на\s*личн.*счет.*предпринимател)`,
			allowIncome:  true,
			allowExpense: true,
		},
		{
			category:     "Возвраты",
			pattern:      `(?i)(возврат|возмещени)`,
			allowIncome:  true,
			allowExpense: true,
		},
		{
			category:     "Налоги и взносы",
			pattern:      `(?i)(енп|единый\s*налог|налог|взнос|усн|ндфл|страхов.*взнос)`,
			allowIncome:  false,
			allowExpense: true,
		},
		{
			category:     "Зарплата",
			pattern:      `(?i)(заработн.*плат|зарплат|аванс|оплат[аы]?\s*труд|окончательн.*расчет|расчет.*сотрудник)`,
			allowIncome:  false,
			allowExpense: true,
		},
		{
			category:     "Коммунальные",
			pattern:      `(?i)(электроэнерг|водоснабжен|водоотведен|отоплени|коммунальн|жкх|газоснабжен|теплоснабжен|вывоз.*тко|уборк)`,
			allowIncome:  false,
			allowExpense: true,
		},
		{
			category:     "Реклама и маркетинг",
			pattern:      `(?i)(рекламн.*кабинет|реклам|маркетинг|продвижени|директ|таргет|смм)`,
			allowIncome:  false,
			allowExpense: true,
		},
		{
			category:     "Офисные и хозтовары",
			pattern:      `(?i)(канцтовар|хозтовар|офисн.*расход)`,
			allowIncome:  false,
			allowExpense: true,
		},
		{
			category:     "Административные расходы",
			pattern:      `(?i)(бухгалтерск|охраны|охран.*услуг|топливн.*карт|юридическ.*услуг)`,
			allowIncome:  false,
			allowExpense: true,
		},
		{
			category:     "Банковские комиссии",
			pattern:      `(?i)(комисси|обслуживан.*счет|рко|вознагражден.*банк)`,
			allowIncome:  false,
			allowExpense: true,
		},
		{
			category:     "Кредиты и займы",
			pattern:      `(?i)(кредит|займ|лизинг|процент|погашени)`,
			allowIncome:  true,
			allowExpense: true,
		},
		{
			category:     "Логистика",
			pattern:      `(?i)(доставк|транспортн|перевозк|груз|логистик)`,
			allowIncome:  false,
			allowExpense: true,
		},
		{
			category:     "Связь и IT",
			pattern:      `(?i)(интернет|связ|мобильн|облачн|хостинг|сервер|подписк|софт|лицензи)`,
			allowIncome:  false,
			allowExpense: true,
		},
		{
			category:     "Аренда",
			pattern:      `(?i)(аренд|помещени|склад|офис)`,
			allowIncome:  false,
			allowExpense: true,
		},
		{
			category:     "Поставщики",
			pattern:      `(?i)(поставщик|сырь|товар|материал|упд|счет[-\s]?фактур)`,
			allowIncome:  false,
			allowExpense: true,
		},
		{
			category:     "Выручка",
			pattern:      `(?i)(выручк|реализаци|поступлен.*покупател|оплат[аы]?\s*от\s*покупател|сбп|зачислен.*по\s*операциям|оплат[аы]?\s*через\s*сбп)`,
			allowIncome:  true,
			allowExpense: false,
		},
	}

	for _, r := range rawRules {
		categoryRules = append(categoryRules, categoryRule{
			category:     r.category,
			regex:        regexp.MustCompile(r.pattern),
			allowIncome:  r.allowIncome,
			allowExpense: r.allowExpense,
		})
	}
}

// ClassifyCategory classifies payment description and direction into a business category
func ClassifyCategory(description string, isIncome bool) string {
	desc := strings.TrimSpace(description)
	if desc == "" {
		return "Прочее"
	}

	for _, rule := range categoryRules {
		if isIncome && !rule.allowIncome {
			continue
		}
		if !isIncome && !rule.allowExpense {
			continue
		}
		if rule.regex.MatchString(desc) {
			return rule.category
		}
	}

	return "Прочее"
}
