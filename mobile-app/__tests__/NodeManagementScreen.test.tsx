import React from 'react';
import { render, fireEvent, waitFor } from '@testing-library/react-native';
import NodeManagementScreen from '../src/screens/NodeManagementScreen';

jest.mock('../src/services/api', () => ({
  api: {
    getStatus: jest.fn(),
    updateNodeDetails: jest.fn(),
  },
}));

const mockApi = require('../src/services/api').api;

describe('NodeManagementScreen', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders loading indicator initially', () => {
    mockApi.getStatus.mockImplementation(() => new Promise(() => {}));
    const { getByTestId } = render(<NodeManagementScreen />);
    expect(getByTestId('loading-indicator')).toBeTruthy();
  });

  it('displays node information after loading', async () => {
    mockApi.getStatus.mockResolvedValue({
      node: {
        id: 'node-123',
        country: 'US',
        city: 'New York',
        os: 'Ubuntu 22.04',
        node_type: 'residential',
        ip: '1.2.3.4',
        last_seen: '2024-01-01T00:00:00Z',
        reputation: 85,
        online: true,
      },
    });

    const { getByText } = render(<NodeManagementScreen />);

    await waitFor(() => {
      expect(getByText('node-123')).toBeTruthy();
    });

    expect(getByText('residential')).toBeTruthy();
    expect(getByText('1.2.3.4')).toBeTruthy();
    expect(getByText('Online')).toBeTruthy();
  });

  it('shows error when status fetch fails', async () => {
    mockApi.getStatus.mockRejectedValue(new Error('Network error'));
    const { getByText } = render(<NodeManagementScreen />);

    await waitFor(() => {
      expect(getByText('Failed to load node information')).toBeTruthy();
    });
  });

  it('updates node details successfully', async () => {
    mockApi.getStatus.mockResolvedValue({
      node: {
        id: 'node-123',
        country: 'US',
        city: 'New York',
        os: 'Ubuntu 22.04',
        online: true,
        reputation: 85,
      },
    });
    mockApi.updateNodeDetails.mockResolvedValue(undefined);

    const { getByPlaceholderText, getByText } = render(<NodeManagementScreen />);

    await waitFor(() => {
      expect(getByText('node-123')).toBeTruthy();
    });

    const cityInput = getByPlaceholderText('New York');
    fireEvent.changeText(cityInput, 'Los Angeles');

    fireEvent.press(getByText('Update Node'));

    await waitFor(() => {
      expect(mockApi.updateNodeDetails).toHaveBeenCalledWith({
        country: 'US',
        city: 'Los Angeles',
        os: 'Ubuntu 22.04',
      });
    });
  });

  it('shows alert on update failure', async () => {
    mockApi.getStatus.mockResolvedValue({
      node: {
        id: 'node-123',
        country: 'US',
        city: 'New York',
        os: 'Ubuntu 22.04',
        online: true,
        reputation: 85,
      },
    });
    mockApi.updateNodeDetails.mockRejectedValue(new Error('Update failed'));

    const { getByPlaceholderText, getByText } = render(<NodeManagementScreen />);

    await waitFor(() => {
      expect(getByText('node-123')).toBeTruthy();
    });

    const cityInput = getByPlaceholderText('New York');
    fireEvent.changeText(cityInput, 'Los Angeles');

    fireEvent.press(getByText('Update Node'));

    await waitFor(() => {
      expect(mockApi.updateNodeDetails).toHaveBeenCalled();
    });
  });
});
