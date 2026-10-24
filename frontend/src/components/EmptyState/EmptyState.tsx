import { ReactNode } from 'react';
import './EmptyState.css';

interface EmptyStateProps {
    /** Эмодзи или иконка. Если ReactNode — рендерится как есть. */
    icon?: ReactNode | string;
    title: string;
    description?: string;
    /** Кнопка действия: "Создать", "Войти", "На главную" и т.п. */
    action?: ReactNode;
    /** Уменьшенный вариант — для встраивания в карточку/секцию */
    compact?: boolean;
    className?: string;
}

const EmptyState = ({
    icon = '📭',
    title,
    description,
    action,
    compact = false,
    className = '',
}: EmptyStateProps) => {
    return (
        <div
            className={`EmptyState ${compact ? 'EmptyState--compact' : ''} ${className}`}
            role="status"
        >
            <div className="EmptyState-icon" aria-hidden="true">
                {icon}
            </div>

            <h3 className="EmptyState-title">{title}</h3>

            {description && (
                <p className="EmptyState-description">{description}</p>
            )}

            {action && <div className="EmptyState-action">{action}</div>}
        </div>
    );
};

export default EmptyState;