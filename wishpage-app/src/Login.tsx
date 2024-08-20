import { useState } from "react";
import { request } from "./request";
import { LoginDialog } from "./ConfirmDialog";

interface LoginProps {
    isAdminMode: boolean
    setIsAdminMode: (mode: boolean) => void
    displayEditButton: boolean
}

async function sha256(message: string): Promise<string> {
    // Encode the message as UTF-8
    const msgBuffer = new TextEncoder().encode(message);

    // Hash the message
    const hashBuffer = await crypto.subtle.digest('SHA-256', msgBuffer);

    // Convert ArrayBuffer to Array
    const hashArray = Array.from(new Uint8Array(hashBuffer));

    // Convert bytes to hex string
    const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('');

    return hashHex;
}

export function Login({ isAdminMode, setIsAdminMode, displayEditButton }: LoginProps): JSX.Element {
    const [isDialogOpen, setIsDialogOpen] = useState(false)

    const handleSubmit = async (password: string): Promise<boolean> => {
        const hashedPassword = await sha256(password)
        try {
            const response = await request("POST", "login", { password: hashedPassword }, true)
            localStorage.setItem("token", response.token)
        } catch (error) {
            console.error("login error ", error)
            return false
        }
        setIsDialogOpen(false)
        setIsAdminMode(true)
        return true
    }

    return <div>
        {
            displayEditButton && <div style={{ marginTop: "20px" }}>
                {!isAdminMode && <button onClick={() => setIsDialogOpen(true)}>Edit</button>}
                {isAdminMode && <button onClick={() => setIsAdminMode(false)}>Edit Complete</button>}
            </div>
        }
        <LoginDialog
            isOpen={isDialogOpen}
            message="Please enter the Admin password"
            onLogin={handleSubmit}
            onCancel={() => setIsDialogOpen(false)}
        />
    </div>
}