import { z } from 'zod';

export const nameSchema = z.object({
    new_name: z
        .string()
        .min(3, 'Минимум 3 символа')
        .max(16, 'Максимум 16 символов')
        .regex(/^[a-z0-9_-]+$/, 'Только латиница, цифры, _ и -'),
});

export type NameFormData = z.infer<typeof nameSchema>;