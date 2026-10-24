import { Link } from "react-router-dom";
import '../App.css';
import EmptyState from "../components/EmptyState";

const NotFoundPage = () => {
    return (
        <div className="NotFoundPage">
            <EmptyState
                icon="🧭"
                title="Страница не найдена"
                description="Возможно, она была удалена, переехала или вы ошиблись в адресе."
                action={
                    <>
                        <Link to="/" className="Button">
                            На главную
                        </Link>
                        <button
                            className="Button"
                            onClick={() => window.history.back()}
                        >
                            Назад
                        </button>
                    </>
                }
            />
            <div className="NotFoundPage-code">404</div>
        </div>
    );
};

export default NotFoundPage;