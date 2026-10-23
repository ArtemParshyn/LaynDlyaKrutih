import { z } from 'zod';

export const nameSchema = z.object({
  new_name: z
    .string()
    .min(2, 'Минимум 2 символа')
    .max(64, 'Максимум 64 символа'),
});

export type NameFormData = z.infer<typeof nameSchema>;