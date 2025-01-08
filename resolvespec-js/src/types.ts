// Types
export type Operator = 'eq' | 'neq' | 'gt' | 'gte' | 'lt' | 'lte' | 'like' | 'ilike' | 'in';
export type Operation = 'read' | 'create' | 'update' | 'delete';
export type SortDirection = 'asc' | 'desc';

export interface PreloadOption {
    relation: string;
    columns?: string[];
    filters?: FilterOption[];
}

export interface FilterOption {
    column: string;
    operator: Operator;
    value: any;
}

export interface SortOption {
    column: string;
    direction: SortDirection;
}

export interface CustomOperator {
    name: string;
    sql: string;
}

export interface ComputedColumn {
    name: string;
    expression: string;
}

export interface Options {
    preload?: PreloadOption[];
    columns?: string[];
    filters?: FilterOption[];
    sort?: SortOption[];
    limit?: number;
    offset?: number;
    customOperators?: CustomOperator[];
    computedColumns?: ComputedColumn[];
}

export interface RequestBody {
    operation: Operation;
    id?: string | string[];
    data?: any | any[];
    options?: Options;
}

export interface APIResponse<T = any> {
    success: boolean;
    data: T;
    metadata?: {
        total: number;
        filtered: number;
        limit: number;
        offset: number;
    };
    error?: {
        code: string;
        message: string;
        details?: any;
    };
}

export interface Column {
    name: string;
    type: string;
    is_nullable: boolean;
    is_primary: boolean;
    is_unique: boolean;
    has_index: boolean;
}

export interface TableMetadata {
    schema: string;
    table: string;
    columns: Column[];
    relations: string[];
}

export interface ClientConfig {
    baseUrl: string;
    token?: string;
}
