import '../../App.css';

interface PlasticCardProps {
    companyName: string; // <-- Обязательный пропс для имени
    ostatok: number;
    cardStatus?: string;
}

const PlasticCard = ({ companyName, ostatok, cardStatus = 'Стабилен' }: PlasticCardProps) => {
    return (
        <div className='plastic-card'>
            {/* Верхний ряд: Логотип и Чип */}
            <div className="card-top-row">
                <div className="card-logo">ЦЕНТР-ИНВЕСТ</div>
                <div className="card-chip"></div>
            </div>

            {/* Центральная часть: Сумма */}
            <div className="card-middle-row">
                <div className="card-label">Остаток по счетам</div>
                <div className="card-amount">
                    {ostatok.toLocaleString('ru-RU')} ₽
                </div>
            </div>

            {/* Нижний ряд: Владелец и Статус */}
            <div className="card-bottom-row">
                <div className="card-holder">
                    <div className="card-label-small">Владелец</div>
                    <div className="card-value">{companyName}</div>
                </div>
                <div className="card-status">
                    <div className="card-label-small">Статус</div>
                    <div className="card-value">{cardStatus}</div>
                </div>
            </div>
        </div>
    );
};

export default PlasticCard;