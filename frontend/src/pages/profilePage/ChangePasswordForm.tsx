import { useState } from 'react';
import { useAuth } from '../../context/AuthContext';
import '../../App.css';
import { toast } from 'sonner';
import ReactModal from 'react-modal';

interface ChangePasswordFormProps {
    isOpen: boolean;
    onClose: () => void;
}

const ChangePasswordForm = ({ isOpen, onClose }: ChangePasswordFormProps) => {
    const [isModalOpen, setModalOpen] = useState(false);
    const { changePassword } = useAuth();
    const [oldPassword, setOldPassword] = useState('');
    const [newPassword, setNewPassword] = useState('');
    const [confirmPassword, setConfirmPassword] = useState('');
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');

    const handleModalOpen = () => {
        setModalOpen(true)
    }
    const isDisabled =
        !oldPassword ||
        !newPassword ||
        !confirmPassword ||
        newPassword !== confirmPassword ||
        loading;

    const handleSubmit = async () => {
        setError('');
        setLoading(true);
        try {
            await changePassword({ oldPassword, newPassword });
            setOldPassword('');
            setNewPassword('');
            setConfirmPassword('');
            toast.success('Пароль успешно изменен');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Ошибка смены пароля');
        } finally {
            setLoading(false);
        }
    }
    const handleChangePass = (event: React.ChangeEvent<HTMLInputElement>) => {
        setError('')
        const target = event.target
        const name = target.name

        if (name === 'old_pass') setOldPassword(target.value)
        if (name === 'new_pass') setNewPassword(target.value)
        if (name === 'confirm_pass') setConfirmPassword(target.value)
    }
    return (

        <div>
            <ReactModal
                shouldCloseOnOverlayClick={false}
                shouldCloseOnEsc={false}
                isOpen={isOpen} onRequestClose={onClose}
                className='RegistrationModal'
                overlayClassName='RegistrationModal-overlay'
            >
                <button className='Button'
                    onClick={() => onClose()}>

                    ❌
                </button>
                <input type="password" placeholder='Старый пароль' name='old_pass' value={oldPassword} onChange={handleChangePass} />
                <input type="password" placeholder='Новый пароль' name='new_pass' value={newPassword} onChange={handleChangePass} />
                <input type="password" placeholder='Подтвердите новый пароль' name='confirm_pass' value={confirmPassword} onChange={handleChangePass} />
                {error && <div className="error-message">{error}</div>}
                <button className='Button' onClick={handleSubmit} disabled={isDisabled}>
                    Подтвердить
                </button>

            </ReactModal>
        </div>
    );
}

export { ChangePasswordForm };