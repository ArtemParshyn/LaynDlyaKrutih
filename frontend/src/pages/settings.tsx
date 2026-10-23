import EmptyState from '../components/EmptyState';

const SettingsPage = () => {
    return (
        <div>
            <h1 className="PageTitle">Настройки</h1>
            <EmptyState
                icon="🛠️"
                title="Раздел в разработке"
                description="Скоро здесь появятся настройки уведомлений, приватности и другие. Пока что можете изменить имя, аватар и пароль на странице профиля."
                action={
                    <a href="/profile" className="Button">Перейти в профиль</a>
                }
            />
        </div>
    );
};

export default SettingsPage;