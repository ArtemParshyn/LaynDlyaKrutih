import { useState } from 'react';
import { useAuth } from '../../context/AuthContext';
import '../../App.css';
import { toast } from 'sonner';
import ReactModal from 'react-modal';
import { CircleLoader } from 'react-spinners';

interface ChangePasswordFormProps {
  isOpen: boolean;
  onClose: () => void;
}

const ChangePasswordForm = ({ isOpen, onClose }: ChangePasswordFormProps) => {
  const { changePassword } = useAuth();
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const isDisabled =
    !newPassword ||
    newPassword.length < 6 ||
    !confirmPassword ||
    newPassword !== confirmPassword ||
    loading;

  const handleSubmit = async () => {
    if (newPassword.length < 6) {
      setError('Пароль должен содержать не менее 6 символов');
      return;
    }
    if (newPassword !== confirmPassword) {
      setError('Пароли не совпадают');
      return;
    }

    setError('');
    setLoading(true);
    try {
      await changePassword(newPassword);
      setNewPassword('');
      setConfirmPassword('');
      toast.success('Пароль успешно изменен!');
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка смены пароля');
    } finally {
      setLoading(false);
    }
  };

  const handleChangePass = (event: React.ChangeEvent<HTMLInputElement>) => {
    setError('');
    const target = event.target;
    if (target.name === 'new_pass') setNewPassword(target.value);
    if (target.name === 'confirm_pass') setConfirmPassword(target.value);
  };

  return (
    <ReactModal
      shouldCloseOnOverlayClick={false}
      shouldCloseOnEsc={false}
      isOpen={isOpen}
      onRequestClose={onClose}
      className="RegistrationModal"
      overlayClassName="RegistrationModal-overlay"
    >
      <button className="Button" onClick={onClose}>
        ❌
      </button>
      <h3>Смена пароля</h3>
      <input
        type="password"
        placeholder="Новый пароль (минимум 6 символов)"
        name="new_pass"
        value={newPassword}
        onChange={handleChangePass}
        autoComplete="new-password"
      />
      <input
        type="password"
        placeholder="Подтвердите новый пароль"
        name="confirm_pass"
        value={confirmPassword}
        onChange={handleChangePass}
        autoComplete="new-password"
      />
      {error && <div className="error-message">{error}</div>}
      <button className="Button" onClick={handleSubmit} disabled={isDisabled}>
        {loading ? <CircleLoader color="var(--text)" size={18} /> : 'Подтвердить'}
      </button>
    </ReactModal>
  );
};

export { ChangePasswordForm };