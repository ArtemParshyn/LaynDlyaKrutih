"""Таксономия категорий и справочник ролей контрагентов.

Категории спроектированы под решения малого бизнеса, а не под бухгалтерский
план счетов: выручка -> себестоимость -> постоянные -> переменные -> налоги ->
кредиты -> деньги владельца.

Правила здесь описывают **контрагента**, а не текст назначения платежа:
роль контрагента стабильна, а формулировки меняются от платежа к платежу.
"""

from __future__ import annotations

# --- Категории ---------------------------------------------------------
# group:      operating / financing / investing / owner / correction / unknown
# flow:       inflow / outflow / both
# behavior:   fixed / variable / one_off
# mandatory:  платёж нельзя не сделать (аренда, зарплата, налоги, кредит)

CATEGORIES: dict[str, dict] = {
    "Выручка": {
        "group": "operating", "flow": "inflow", "behavior": "variable", "mandatory": False,
        "description": "Деньги от клиентов: эквайринг, СБП, счета, маркетплейсы, госзакупки",
    },
    "Себестоимость": {
        "group": "operating", "flow": "outflow", "behavior": "variable", "mandatory": False,
        "description": "Товары, сырьё, материалы, упаковка — то, что прямо связано с выручкой",
    },
    "Аренда": {
        "group": "operating", "flow": "outflow", "behavior": "fixed", "mandatory": True,
        "description": "Помещения, склады, POS-терминалы",
    },
    "Зарплата": {
        "group": "operating", "flow": "outflow", "behavior": "fixed", "mandatory": True,
        "description": "Сотрудники и подрядчики по ГПХ",
    },
    "Налоги": {
        "group": "operating", "flow": "outflow", "behavior": "variable", "mandatory": True,
        "description": "ЕНП, УСН, НДФЛ, страховые взносы",
    },
    "Коммунальные": {
        "group": "operating", "flow": "outflow", "behavior": "fixed", "mandatory": True,
        "description": "Электроэнергия, вода, вывоз ТКО",
    },
    "Связь и IT": {
        "group": "operating", "flow": "outflow", "behavior": "fixed", "mandatory": False,
        "description": "Телеком, SaaS, хостинг, поддержка",
    },
    "Маркетинг": {
        "group": "operating", "flow": "outflow", "behavior": "variable", "mandatory": False,
        "description": "Реклама и продвижение — первое, что режут при кассовом разрыве",
    },
    "Логистика": {
        "group": "operating", "flow": "outflow", "behavior": "variable", "mandatory": False,
        "description": "Доставка, экспедирование, топливо, платные дороги",
    },
    "Административные": {
        "group": "operating", "flow": "outflow", "behavior": "fixed", "mandatory": False,
        "description": "Бухгалтерия, охрана, клининг, юристы, банковские комиссии, ремонт",
    },
    "Кредиты и лизинг": {
        "group": "financing", "flow": "outflow", "behavior": "fixed", "mandatory": True,
        "description": "Тело кредита, проценты, лизинговые платежи",
    },
    "Оборудование": {
        "group": "investing", "flow": "outflow", "behavior": "one_off", "mandatory": False,
        "description": "Техника, мебель, оборудование — разовые вложения",
    },
    "Внутренние и владелец": {
        "group": "owner", "flow": "both", "behavior": "one_off", "mandatory": False,
        "description": "Свои счета, личные выплаты, займы учредителя — не выручка и не расход",
    },
    "Возвраты": {
        "group": "correction", "flow": "both", "behavior": "one_off", "mandatory": False,
        "description": "Возвраты клиентам и от поставщиков — корректировка, не доход/расход",
    },
    "Прочее": {
        "group": "unknown", "flow": "both", "behavior": "one_off", "mandatory": False,
        "description": "Не удалось определить по контрагенту и направлению",
    },
}

# --- Роли контрагентов (нормализованное имя без формы и номера) --------

BANK = {"Банк"}
TAXES = {"Казначейство России (ФНС России)"}
EMPLOYEE = {"Сотрудник"}
FOUNDER = {"Учредитель (физическое лицо)"}
OWNER = {"Предприниматель", "Участник общества (физическое лицо)"}
SELF_EMPLOYED = {"Физическое лицо (самозанятый)"}
INDIVIDUAL = {"Физическое лицо"}
RENT = {"Арендодатель"}
UTILITIES = {"Водоканал", "Энергосбыт", "Региональный оператор ТКО"}
TELECOM_IT = {"Телеком", "Учетные системы", "Облачный провайдер", "Софт-Решения"}
MARKETING = {"Рекламная платформа"}
LOGISTICS = {
    "Транспортная компания", "Экспедитор", "Курьерская служба", "Логистика",
    "Грузоперевозки", "Такси-Парк", "Каршеринг", "Стройтранс",
    "Система взимания платы", "Фулфилмент-центр", "Топливная компания",
}
CAPEX = {"Оборудование-Снаб", "Холодильное оборудование", "Техника-Трейд", "Салонное оборудование"}
LEASING = {"Лизинговая компания"}
INSURANCE = {"Страховая компания"}
ADMIN = {
    "Бухгалтерия-Аутсорс", "Охранное предприятие", "Клининг", "Клининг-Сервис",
    "Истец", "Бассейн-Сервис", "Ремонт-Строй", "Фасад-Сервис", "Сантехмонтаж",
    "Электромонтаж", "Отделка-Профи", "Стройподряд", "Генподрядчик",
}
OFFICE = {"Канцтовары", "Хозтовары"}
MED_LAB = {"Лаборатория анализов", "Зуботехническая лаборатория"}
AUTO_SERVICE = {"Автоцентр", "Грузовой автосервис", "Шиномонтаж-Сервис"}

FOOD_SUPPLIERS = {
    "Продукты-Опт", "Мясокомбинат", "Кондитерская фабрика", "Молочный комбинат",
    "Хлебозавод", "Овощебаза", "Бакалея-Опт", "Напитки-Дистрибуция",
    "Заморозка-Дистрибуция", "Агрокомбинат", "Кофеобжарка", "Продукты",
    "Мукомольный комбинат", "Маслозавод", "Зернотрейдер", "Экспорт-Зерно",
    "Элеватор", "Агрохолдинг", "Агро", "Масла-Дистрибуция",
}
OTHER_SUPPLIERS = {
    "Бытовая химия-Опт", "Автозапчасти-Опт", "Автохимия", "Инструмент-Снаб",
    "Стройматериалы", "Кровля-Снаб", "ЛДСП-Опт", "Металлобаза", "Метизы",
    "Фурнитура", "Электротехника-Снаб", "Электроника-Опт", "Лакокрасочный завод",
    "Бетонный завод", "Лесоторговая база", "Текстиль-Опт", "Аксессуары-Опт",
    "Трикотаж", "Обувная фабрика", "Швейная фабрика", "Мебельная фабрика",
    "Фабрика товаров", "Производство игрушек", "Упаковка-Юг", "Медтехника",
    "Медицинские материалы", "Косметика-Проф", "Расходники для салонов",
    "Запчасти для сельхозтехники", "Запчасти для грузовиков", "Шинный центр",
    "Текстиль для гостиниц", "Семена", "Агрохимия", "Удобрения", "Импортер",
    "Импорт-Трейд", "Завод-изготовитель", "Оптторг", "Отделка-Опт",
    "Элеватор-Услуги",
    # торговые компании могут быть и поставщиками, и клиентами —
    # направление операции решает, что это
    "Завод", "Торговый дом", "Дистрибьютор", "Дистрибуция",
}

GRANT = {"УФК по региону (Фонд поддержки предпринимательства)"}
MARKETPLACE = {"Маркетплейс"}
GOVERNMENT_CUSTOMER_PREFIX = "УФК по региону (МКУ"
GOVERNMENT_CUSTOMER = {"УФК по региону (Министерство сельского хозяйства региона)"}

CUSTOMER = {
    "Супермаркет", "Минимаркет", "Магазин", "Кафе", "Ресторан", "Торговая сеть",
    "Строймаркет", "Ритейл", "Гостиница", "База отдыха", "Клиника", "Кофейня",
    "Модный двор", "Детский центр", "Онлайн-школа", "Медцентр", "Автосервис",
    "Мебельный салон", "Торговый центр", "Федеральная сеть", "Холдинг",
    "Госкорпорация-Партнер", "Офис-Центр", "Управляющая компания", "Застройщик",
    "Девелопмент", "Дистрибьютор", "Дистрибуция", "Торговый дом", "Завод",
    "Онлайн-Товары",
}
CUSTOMER_PREFIX = "Клиент-"

# Особые контрагенты: текст всё ещё нужен, отрасль не решает
SPECIAL_INDUSTRIES = {
    "bank", "taxes", "employee", "founder", "owner", "self_employed",
    "individual", "grant", "government_customer",
}

# Для этих отраслей не применяем сглаживание по контрагенту: внутри одного
# контрагента законно живут разные подкатегории (банк, налоги, физлица).
NO_SMOOTHING = SPECIAL_INDUSTRIES | {"refund"}


def industry_of(role: str, direction: str) -> str:
    """Отраслевая группа контрагента по нормализованному имени.

    Направление участвует только там, где одна и та же роль может быть
    и поставщиком, и клиентом.
    """
    if role in BANK:
        return "bank"
    if role in TAXES:
        return "taxes"
    if role in EMPLOYEE:
        return "employee"
    if role in FOUNDER:
        return "founder"
    if role in OWNER:
        return "owner"
    if role in SELF_EMPLOYED:
        return "self_employed"
    if role in INDIVIDUAL:
        return "individual"
    if role in GRANT:
        return "grant"
    if role in MARKETPLACE:
        return "marketplace"
    if role.startswith(GOVERNMENT_CUSTOMER_PREFIX) or role in GOVERNMENT_CUSTOMER:
        return "government_customer"
    if role in RENT:
        return "rent"
    if role in UTILITIES:
        return "utilities"
    if role in TELECOM_IT:
        return "telecom_it"
    if role in MARKETING:
        return "marketing"
    if role in LOGISTICS:
        return "logistics"
    if role in CAPEX:
        return "capex"
    if role in LEASING:
        return "leasing"
    if role in INSURANCE:
        return "insurance"
    if role in OFFICE:
        return "office"
    if role in MED_LAB:
        return "med_lab"
    if role in AUTO_SERVICE:
        return "auto_service"
    if role in ADMIN:
        return "admin"
    if role in FOOD_SUPPLIERS:
        return "food_supplier"
    if role in OTHER_SUPPLIERS:
        return "other_supplier"
    if role in CUSTOMER or role.startswith(CUSTOMER_PREFIX):
        return "customer"
    return "unknown"
