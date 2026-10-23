
import '../App.css'
import { ProfileButton } from '../registration';
import { Link } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { Toaster, toast } from 'sonner';
import { useEffect } from 'react';
import { useState } from "react";
import ThemeSwap from "../components/Thema/Thema";
import { ReactNode } from "react";
import { useNavigate } from 'react-router-dom';
import Hamburger from "hamburger-react";
import { useLocation } from 'react-router-dom';

const ProfileButtonHeader = () => {
    const { loading, isAuthenticated, openModal } = useAuth();
    if (loading) return null;

    if (!isAuthenticated) {
        return (
            <button
                className='Button'
                onClick={() => toast.error('Пожалуйста войдите в свой аккаунт или зарегестрируйтесь', {
                    action: { label: 'Войти', onClick: () => openModal() }
                })} >
                Мой профиль
            </button>
        );
    }

    return (
        <Link to='/profile' className='Button'> Мой профиль</Link>
    );
}





const SettingsButtonHeader = () => {
    const { loading, isAuthenticated, openModal } = useAuth();
    if (loading) return null;

    if (!isAuthenticated) {
        return (
            <button
                className='Button'
                onClick={() => toast.error('Пожалуйста войдите в свой аккаунт или зарегестрируйтесь', {
                    action: { label: 'Войти', onClick: () => openModal() }
                })} >
                Настройки профиля
            </button>
        );
    }

    return (
        <Link to='/settings' className='Button'> Настройки профиля</Link>
    );
}

const BurgerMenu = () => {
    const navigate = useNavigate();
    const [isOpen, setIsOpen] = useState(false);
    const location = useLocation();
    const handleClick = () => {
        setIsOpen(true);
    };
    useEffect(() => {
        setIsOpen(false);
    }, [location.pathname]);
    return (
        <div className="BurgerMenu">
            <Hamburger toggled={isOpen} toggle={setIsOpen} />

            {isOpen && (
                <div className="BurgerMenu-dropdown">
                    <ProfileButtonHeader />
                    <SettingsButtonHeader />
                </div>
            )}
        </div>
    );
};
const Header = () => {


    return (
        <header className="Header">
            <Link to='/' className="Button">Главная </Link>
            <div className="Header-desktop-only"><ProfileButtonHeader /></div>
            <div className="Header-desktop-only"><SettingsButtonHeader /></div>

            <div className="Header-right-group">
                <ProfileButton />
                <BurgerMenu />
                <ThemeSwap />
            </div>

        </header>
    );
};


export default Header;