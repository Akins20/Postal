import { z } from "zod";

/** Auth form validation - mirrors the backend rules (the server re-validates
 *  and stays authoritative). Ported verbatim from the web. */
const email = z.email("Enter a valid email address");
const newPassword = z.string().min(8, "Use at least 8 characters");

export const loginSchema = z.object({
  email,
  password: z.string().min(1, "Password is required"),
});
export const signupSchema = z.object({ email, password: newPassword });
export const requestResetSchema = z.object({ email });
export const confirmResetSchema = z.object({ new_password: newPassword });

export type LoginValues = z.infer<typeof loginSchema>;
export type SignupValues = z.infer<typeof signupSchema>;
export type RequestResetValues = z.infer<typeof requestResetSchema>;
export type ConfirmResetValues = z.infer<typeof confirmResetSchema>;
