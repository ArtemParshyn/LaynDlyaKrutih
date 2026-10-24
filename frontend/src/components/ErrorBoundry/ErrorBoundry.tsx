import { Component, ErrorInfo, ReactNode } from 'react';
import '../../App.css';


interface ErrorBoundaryProps {
    children: ReactNode;        // ← что оборачиваем
    fallback?: ReactNode;       // ← кастомный UI (опционально)
}

interface ErrorBoundaryState {
    hasError: boolean;
}

class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
    state: ErrorBoundaryState = { hasError: false };

    static getDerivedStateFromError(): ErrorBoundaryState {
        return { hasError: true };
    }

    componentDidCatch(error: Error, errorInfo: ErrorInfo) {
        console.error('ErrorBoundary поймал:', error, errorInfo);
    }

    render() {
        if (this.state.hasError) {
            return (
                <div>

                    <h1>Что-то пошло не так</h1>
                    <button onClick={() => window.location.reload()}>
                        Перезагрузить
                    </button>
                </div>
            );
        }
        return this.props.children;
    }
}

export default ErrorBoundary;