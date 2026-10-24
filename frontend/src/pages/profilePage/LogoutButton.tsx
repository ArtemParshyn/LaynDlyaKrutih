import { useAuth } from "../../context/AuthContext";
import { useState } from "react";
import { toast } from "sonner";

interface LogoutButtonProps {
  compact?: boolean;
}

const DoorIcon = () => (
  <svg
    width="16"
    height="16"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <path d="M13 4h3a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2h-3" />
    <path d="M10 17l5-5-5-5" />
    <line x1="15" y1="12" x2="3" y2="12" />
  </svg>
);

const LogoutButton = ({ compact = false }: LogoutButtonProps) => {
  const { logout } = useAuth();
  const [loading, setLoading] = useState(false);

  const handleLogout = async () => {
    setLoading(true);
    try {
      await logout();
      toast.success('Вы вышли из аккаунта');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Ошибка выхода');
    } finally {
      setLoading(false);
    }
  };

  return (
    <button
      className={compact ? 'ProfileCard-logout' : 'Button'}
      onClick={handleLogout}
      disabled={loading}
      title="Выйти из аккаунта"
    >
      {compact ? <DoorIcon /> : 'Выйти из аккаунта'}
    </button>
  );
};

export default LogoutButton;