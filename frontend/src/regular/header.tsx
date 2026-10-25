import '../App.css';
import { ProfileButton } from '../registration';
import { Link } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { toast } from 'sonner';
import { useEffect, useState } from 'react';
import ThemeSwap from "../components/Thema/Thema";
import Hamburger from "hamburger-react";
import { useLocation } from 'react-router-dom';

const ProfileButtonHeader = () => {
  const { loading, isAuthenticated, openModal } = useAuth();
  if (loading) return null;

  if (!isAuthenticated) {
    return (
      <button
        className="Button"
        onClick={() =>
          toast.error('Пожалуйста, войдите в свой аккаунт или зарегистрируйтесь', {
            action: { label: 'Войти', onClick: () => openModal() },
          })
        }
      >
        Мой профиль
      </button>
    );
  }

  return (
    <Link to="/profile" className="Button">
      Мой профиль
    </Link>
  );
};

const BurgerMenu = () => {
  const [isOpen, setIsOpen] = useState(false);
  const location = useLocation();

  useEffect(() => {
    setIsOpen(false);
  }, [location.pathname]);

  return (
    <div className="BurgerMenu">
      <Hamburger toggled={isOpen} toggle={setIsOpen} label="Меню" />

      {isOpen && (
        <div className="BurgerMenu-dropdown">
          <ProfileButtonHeader />
        </div>
      )}
    </div>
  );
};

const Header = () => {
  return (
    <header className="Header">
      <Link to="/" className="Brand">
        <span className="Brand-mark" aria-hidden="true">Ц</span>
        <span className="Brand-copy">
          <strong>Центр-инвест</strong>
          <small>Финансовая аналитика</small>
        </span>
      </Link>
      <div className="Header-right-group">
        <div className="Header-icon-stack">
          <ProfileButton />
          <ThemeSwap />
        </div>
        <BurgerMenu />
      </div>
    </header>
  );
};

export default Header;
