
import { useAuth } from '../../context/AuthContext'
import ReactModal from 'react-modal';
import '../../App.css';
import React, { useState } from "react";
import { toast } from "sonner";
import { CircleLoader } from "react-spinners";
import { useForm, SubmitHandler } from "react-hook-form"

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
    } = useForm<NameFormData>({
        resolver: zodResolver(nameSchema),
    });
    const { user, updateName } = useAuth();
    const [modalOpen, setModalOpen] = useState(false);


    const onSubmit = async (data: NameFormData) => {
        try {

            await updateName(data.new_name);
            toast.success('Имя профиля успешно изменено)')
            onClose();
        } catch (err) {
            setError('root', { message: err instanceof Error ? err.message : 'Ошибка смены имени' });

        }
    }

    return (
        <div>
            <ReactModal isOpen={isOpen}
                onRequestClose={onClose}
                shouldCloseOnOverlayClick={false}
                shouldCloseOnEsc={false}
                className={'RegistrationModal'} overlayClassName='RegistrationModal-overlay'>

                <button className='Button' onClick={() => onClose()}>
                    ❌
                </button>


                <form onSubmit={handleSubmit(onSubmit)}>
                    <input {...register('new_name')} placeholder="Новое имя" />

                    <button type='submit' className='Button' disabled={isSubmitting}>
                        {isSubmitting ? <CircleLoader color={'#fff'} size={18} /> : 'Подтвердить'}
                    </button>
                    {errors.new_name && (
                        <div className="error-message">{errors.new_name.message}</div>
                    )}

                </form>
            </ReactModal>

        </div>
    );

}
export default EditNameForm;
