# Saga

![Chaos](https://miro.medium.com/v2/resize:fit:1400/0*ef5vZOr1oEDVpi1e)

In the realm of microservices and distributed systems, maintaining data consistency across multiple services can be a daunting challenge. Traditional ACID transactions are often inadequate in such environments due to their inherent limitations when it comes to scalability and network latency. https://www.linkedin.com/pulse/saga-design-pattern-cnet-amir-doosti/

Sagas can be implemented in **“two ways”**:
1. **Orchestration (centralized)**: In this approach, a single orchestrator service manages the entire saga workflow. It explicitly tells each participating service what local transaction to perform and when, tracks the saga state, and handles failure recovery by triggering compensating transactions if needed. This central coordinator has knowledge of the whole saga flow, making it easier to manage complex workflows and reduce cyclic dependencies between services. However, it introduces a single point of failure and adds design complexity because the orchestrator must maintain coordination logic.

2. **Choreography (decentralized)**: Here, there is no central coordinator. Instead, each service performs its local transaction and publishes domain events that trigger the next steps in other services. Each service knows how to react to incoming events independently. This reduces coupling and avoids a single point of failure. It's simpler for small or straightforward workflows but can become confusing and difficult to manage as the saga grows, and integration testing is more challenging since all services need to run together.

## Key Concepts
1. **Saga**: A Saga represents a sequence of distributed transactions or operations across multiple services, where each step is a local transaction. If any step fails, the Saga executes compensating transactions to undo the preceding steps and maintain consistency.
2. **Action**: An Action is an individual step or transaction within a Saga. It performs a specific business operation within a service as part of the overall Saga workflow. Actions can be manually coordinated by explicitly managing their order and flow through handling references to the `Prev()` and `Next()` actions.
3. **Compensate**: A Compensate action is used to reverse the effects of a previous Action in case of failure or rollback.
4. **Execute**: To Execute means to carry out an Action within the Saga workflow. This involves performing the corresponding local transaction as part of advancing the Saga.

## Usage

```go
var subscriptionPlanCreated *models.SubscriptionPlan

// Saga
s := saga.NewSaga("create_subscription_plan_saga", &saga.SagaOptions{
	MaxRetries:    3,
	RetryWaitTime: 1 * time.Second,
})
s.AddAction("subscription_plan_creation",
	func() error {
		subscriptionPlanCreated, err = c.postgresRepository.CreateSubscriptionPlan(ctx, subscriptionPlan)
		if err != nil {
			return err
		}

		return nil
	},
	func() error {
		// Delete stripe plan was previously created
		_, err := c.stripeProvider.RemovePlan(
			stripePlanCreated.ID,
			stripePlanCreated.Product.ID,
			&stripe.PlanParams{},
		)
		if err != nil {
			return err
		}

		c.log.Info("Compensating Action, deleting stripe plan...")
		return nil
	})

err = s.Execute()
if err != nil {
	c.log.Error("Error on executing saga: ", err)
	return nil, err
}
```
