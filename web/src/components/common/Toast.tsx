import { toast as sonnerToast } from 'sonner';
import { CircleCheck, CircleX, AlertTriangle, Info, Loader2 } from 'lucide-react';

type ToastOptions = {
    description?: string;
    duration?: number;
};

const icons = {
    success: <CircleCheck className="size-5 text-primary" />,
    error: <CircleX className="size-5 text-destructive" />,
    warning: <AlertTriangle className="size-5 text-destructive/70" />,
    info: <Info className="size-5 text-accent" />,
    loading: <Loader2 className="size-5 text-muted-foreground animate-spin" />,
};

 export const toast = {
     success: (message: string, options?: ToastOptions) => {
         sonnerToast(message, {
             icon: icons.success,
             position: 'top-left',
             description: options?.description,
             duration: options?.duration,
         });
     },
     error: (message: string, options?: ToastOptions) => {
         sonnerToast(message, {
             icon: icons.error,
             position: 'top-left',
             description: options?.description,
             duration: options?.duration,
         });
     },
     warning: (message: string, options?: ToastOptions) => {
         sonnerToast(message, {
             icon: icons.warning,
             position: 'top-left',
             description: options?.description,
             duration: options?.duration,
         });
     },
     info: (message: string, options?: ToastOptions) => {
         sonnerToast(message, {
             icon: icons.info,
             position: 'top-left',
             description: options?.description,
             duration: options?.duration,
         });
     },
     loading: (message: string, options?: ToastOptions) => {
         return sonnerToast(message, {
             icon: icons.loading,
             duration: Infinity,
             position: 'top-left',
             description: options?.description,
         });
     },
     dismiss: sonnerToast.dismiss,
 };

