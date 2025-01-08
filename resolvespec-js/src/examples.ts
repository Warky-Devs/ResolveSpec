import { getMetadata, read, create, update, deleteEntity } from "./api";
import { ClientConfig } from "./types";

// Usage Examples
const config: ClientConfig = {
    baseUrl: 'http://api.example.com/v1',
    token: 'your-token-here'
};

// Example usage
const examples = async () => {
    // Get metadata
    const metadata = await getMetadata(config, 'test', 'employees');
    

    // Read with relations
    const employees = await read(config, 'test', 'employees', undefined, {
        preload: [
            {
                relation: 'department',
                columns: ['id', 'name']
            }
        ],
        filters: [
            {
                column: 'status',
                operator: 'eq',
                value: 'active'
            }
        ]
    });

    // Create single record
    const newEmployee = await create(config, 'test', 'employees', {
        first_name: 'John',
        last_name: 'Doe',
        email: 'john@example.com'
    });

    // Bulk create
    const newEmployees = await create(config, 'test', 'employees', [
        {
            first_name: 'Jane',
            last_name: 'Smith',
            email: 'jane@example.com'
        },
        {
            first_name: 'Bob',
            last_name: 'Johnson',
            email: 'bob@example.com'
        }
    ]);

    // Update single record
    const updatedEmployee = await update(config, 'test', 'employees', 
        { status: 'inactive' },
        'emp123'
    );

    // Bulk update
    const updatedEmployees = await update(config, 'test', 'employees',
        { department_id: 'dept2' },
        ['emp1', 'emp2', 'emp3']
    );

    // Delete
    await deleteEntity(config, 'test', 'employees', 'emp123');
};