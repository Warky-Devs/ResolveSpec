import { ClientConfig, APIResponse, TableMetadata, Options, RequestBody } from "./types";

// Helper functions
const getHeaders = (options?: Record<string,any>): HeadersInit => {
    const headers: HeadersInit = {
        'Content-Type': 'application/json',
    };

    if (options?.token) {
        headers['Authorization'] = `Bearer ${options.token}`;
    }

    return headers;
};

const buildUrl = (config: ClientConfig, schema: string, entity: string, id?: string): string => {
    let url = `${config.baseUrl}/${schema}/${entity}`;
    if (id) {
        url += `/${id}`;
    }
    return url;
};

const fetchWithError = async <T>(url: string, options: RequestInit): Promise<APIResponse<T>> => {
    try {
        const response = await fetch(url, options);
        const data = await response.json();

        if (!response.ok) {
            throw new Error(data.error?.message || 'An error occurred');
        }

        return data;
    } catch (error) {
        throw error;
    }
};

// API Functions
export const getMetadata = async (
    config: ClientConfig,
    schema: string,
    entity: string
): Promise<APIResponse<TableMetadata>> => {
    const url = buildUrl(config, schema, entity);
    return fetchWithError<TableMetadata>(url, {
        method: 'GET',
        headers: getHeaders(config),
    });
};

export const read = async <T = any>(
    config: ClientConfig,
    schema: string,
    entity: string,
    id?: string,
    options?: Options
): Promise<APIResponse<T>> => {
    const url = buildUrl(config, schema, entity, id);
    const body: RequestBody = {
        operation: 'read',
        options,
    };

    return fetchWithError<T>(url, {
        method: 'POST',
        headers: getHeaders(config),
        body: JSON.stringify(body),
    });
};

export const create = async <T = any>(
    config: ClientConfig,
    schema: string,
    entity: string,
    data: any | any[],
    options?: Options
): Promise<APIResponse<T>> => {
    const url = buildUrl(config, schema, entity);
    const body: RequestBody = {
        operation: 'create',
        data,
        options,
    };

    return fetchWithError<T>(url, {
        method: 'POST',
        headers: getHeaders(config),
        body: JSON.stringify(body),
    });
};

export const update = async <T = any>(
    config: ClientConfig,
    schema: string,
    entity: string,
    data: any | any[],
    id?: string | string[],
    options?: Options
): Promise<APIResponse<T>> => {
    const url = buildUrl(config, schema, entity, typeof id === 'string' ? id : undefined);
    const body: RequestBody = {
        operation: 'update',
        id: typeof id === 'string' ? undefined : id,
        data,
        options,
    };

    return fetchWithError<T>(url, {
        method: 'POST',
        headers: getHeaders(config),
        body: JSON.stringify(body),
    });
};

export const deleteEntity = async (
    config: ClientConfig,
    schema: string,
    entity: string,
    id: string
): Promise<APIResponse<void>> => {
    const url = buildUrl(config, schema, entity, id);
    const body: RequestBody = {
        operation: 'delete',
    };

    return fetchWithError<void>(url, {
        method: 'POST',
        headers: getHeaders(config),
        body: JSON.stringify(body),
    });
};
