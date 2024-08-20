export const request = async (method: string, path: string, data?: object, parseResult: boolean = true, token: string = '') => {
    const options: RequestInit = {
        method: method,
    }
    if (token.length > 0) {
        options.headers = {
            'Content-Type': 'application/json',
            'Authorization': 'Bearer ' + token,
        }
    } else {
        options.headers = {
            'Content-Type': 'application/json',
        }
    }
    if (data) {
        options.body = JSON.stringify(data)
    }
    const response = await fetch(`/${path}`, options);

    if (!response.ok) {
        throw new Error('Network response was not ok');
    }

    if (parseResult) {
        const result = await response.json();
        console.log('Success:', result);
        return result;
    }
    console.log('Success');
    return undefined
};