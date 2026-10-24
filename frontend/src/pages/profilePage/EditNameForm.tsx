import { useAuth } from '../../context/AuthContext';
import ReactModal from 'react-modal';
import '../../App.css';
import { toast } from 'sonner';
import { CircleLoader } from 'react-spinners';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { nameSchema, NameFormData } from '../../schema/nameSchema';

interface ChangeNameFormProps {
  isOpen: boolean;
  onClose: () => void;
}

const EditNameForm = ({ isOpen, onClose }: ChangeNameFormProps) => {
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
    reset,
  } = useForm<NameFormData>({
    resolver: zodResolver(nameSchema),
  });

  const { updateName } = useAuth();

  const onSubmit = async (data: NameFormData) => {
    try {
      await updateName(data.new_name);
      toast.success('Название компании успешно изменено!');
      reset();
      onClose();
    } catch (err) {
      setError('root', {
        message: err instanceof Error ? err.message : 'Ошибка смены имени/компании',
      });
    }
  };

  return (
    <ReactModal
      isOpen={isOpen}
      onRequestClose={onClose}
      shouldCloseOnOverlayClick={false}
      shouldCloseOnEsc={false}
      className="RegistrationModal"
      overlayClassName="RegistrationModal-overlay"
    >
      <button type="button" className="ModalClose" onClick={onClose} aria-label="Закрыть">
        Закрыть
      </button>

      <form onSubmit={handleSubmit(onSubmit)}>
        <h3>Смена названия компании</h3>
        <input
          {...register('new_name')}
          placeholder="Новое название компании (2-64 символов)"
          maxLength={64}
        />

        <button type="submit" className="Button" disabled={isSubmitting}>
          {isSubmitting ? <CircleLoader color="var(--text)" size={18} /> : 'Подтвердить'}
        </button>

        {errors.new_name && (
          <div className="error-message">{errors.new_name.message}</div>
        )}
        {errors.root && (
          <div className="error-message">{errors.root.message}</div>
        )}
      </form>
    </ReactModal>
  );
};

export default EditNameForm;
